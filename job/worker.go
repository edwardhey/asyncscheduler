package job

import (
	"context"
	"encoding/json"
	"fmt"
	"golib/link/httplink"
	"io/ioutil"
	"time"

	"github.com/boltdb/bolt"

	"net/http"

	"edwardhey.com/asyncscheduler/interfaces"
	"edwardhey.com/asyncscheduler/utils"
)

type workerJobTask struct {
	job *Job
	// bucket *bolt.Bucket
}

type workerJobPool struct {
	tasks    <-chan workerJobTask
	poolSize int //启动goroutine的数目
}

func (p *workerJobPool) DoJob(task *workerJobTask) error {
	defer func() {
		//callback通知
		if task.job.Callback != "" && (task.job.Status == StatusFailed || task.job.Status == StatusFinished) {

			_callbackReq := &interfaces.CallbackRequest{
				// Code:         0,
				AttemptTimes: task.job.AttemptTimes,
			}
			if task.job.Status == StatusFailed {
				_callbackReq.Code = 1
			} else {
				_callbackReq.Code = 0
			}
			// v.Add("",task.job.)
			resp, err := httplink.Post(task.job.Callback).ParamsFromStruct(_callbackReq).DoRequest()

			if err != nil || resp.StatusCode != 200 {
				task.job.IsCallbackSuccess = false

				if err != nil {

					utils.InfoWithLableAndFormat("Consume Task", "job:%s callback notify failed err:%s", task.job.ID, err)
				} else {
					defer resp.Body.Close()
					utils.InfoWithLableAndFormat("Consume Task", "job:%s callback notify failed status:%s", task.job.ID, resp.Status)
				}

			} else {
				// fmt.Println(task.job.URL, err, resp)
				task.job.IsCallbackSuccess = true
				utils.InfoWithLableAndFormat("Consume Task", "job:%s callback notify success", task.job.ID)
			}
			// req.ToJSON()
		}
		//save to db
		err := db.GetHandler().Update(func(tx *bolt.Tx) error {
			return task.job.Save(tx)
		})
		if err != nil {
			utils.ErrorWithLableAndFormat("Consume Task", "job:%s save to db error:%v", task.job.ID, err)
		}
	}()

	//执行任务
	var err error
	var req *httplink.Request
	var resp *http.Response

	db.GetHandler().Update(func(tx *bolt.Tx) error {
		if task.job.Status != 0 {
			task.job.Status = StatusRetrying
			task.job.Save(tx)
		}
		// task.job.Status = StatusRetrying
		return nil
	})

	task.job.Mtime = getNowTimestamp()
	task.job.AttemptTimes++
	if task.job.Payload != nil {
		req = httplink.Get(task.job.URL)
	} else {
		req = httplink.Post(task.job.URL).ParamsFromStruct(task.job.Payload)
	}

	resp, err = req.SetTimeout(time.Duration(3)*time.Second, time.Second*time.Duration(10)).DoRequest()
	if err != nil {
		task.job.Failed(err, nil)
		return err
	}
	defer resp.Body.Close()
	if task.job.IgnoreResponse {
		if resp.StatusCode != 200 {
			err := fmt.Errorf("response code:%d", resp.StatusCode)
			task.job.Failed(err, nil)
			return err
		}
		utils.InfoWithLableAndFormat("Consume Task", "job id:%s success", task.job.ID)
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			utils.InfoWithLable("Consume Task", "job:%s read body error:%v", err)
		}

		// var respObject interfaces.Resp
		respObject := &interfaces.Resp{}
		respObject.Code = interfaces.RespStatusErrorMissingAttributes
		// err = req.ToJSON(respObject)
		err = json.Unmarshal(body, respObject)
		if err != nil {
			utils.DebugWithLableAndFormat("Consume Task", "job:%s response json decode failed:%s", task.job.ID, err)
			task.job.Failed(err, nil)
			return err
			// return failed(err, nil)
		}
		if respObject.Code != 0 {
			utils.ErrorWithLableAndFormat("Consume Task", "job:%s response error:%s", task.job.ID, string(body))

			if respObject.Data.AttemptInterval != 0 {
				task.job.AttemptInterval = respObject.Data.AttemptInterval
			}

			if respObject.Err != "" {
				err := fmt.Errorf("code:%d err:%v", respObject.Code, respObject.Err)
				task.job.Failed(err, respObject)
				return err
			}
			err := fmt.Errorf("code:%d err:unknown error", respObject.Code)
			task.job.Failed(err, respObject)
			return err
		}
	}
	task.job.Success()
	//执行通知接口

	return nil
}

func (p *workerJobPool) Run() {
	// var wg sync.WaitGroup
	utils.InfoWithLableAndFormat("worker pool", "%d worker ready!", p.poolSize)
	for i := 0; i < p.poolSize; i++ {
		// wg.Add(1)
		go func() {
			for task := range p.tasks {
				p.DoJob(&task)
			}
			// wg.Done()
		}()
	}

	// wg.Wait()
	// fmt.Println("WorkerPool | Pool exit.")
}

func createTimerWorker(now uint32, _time uint32, interval uint32) {
	// done := make(chan struct{})
	// utils.Debug("create timer", "create worker %d", _time)
	sm.Create(_time)
	if now == _time {
		sm.SendSignal(_time)
	}
	ctx := context.Background()
	smn, _ := sm.Get(_time)
	ctx = context.WithValue(ctx, contextKey("_time"), _time)
	ctx, cancel := context.WithDeadline(ctx, time.Unix(int64(_time+3), 0))
	defer func() {
		// fmt.Println(getNowTimestamp(), _time, "资源回收")
		// utils.Debug("create timer", "gc %d", _time)
		cancel()
		// close(done)
		// sm.Delete(_time)
		sm.Delete(_time)
	}()
	go timerClock(ctx)
	select {
	case <-smn.Done:
	case <-ctx.Done():
		return
	}
}

func timerClock(ctx context.Context) {
	_time := ctx.Value(contextKey("_time")).(uint32)
	smn, err := sm.Get(_time)
	if err != nil {
		return
	}
	for {
		select {
		case <-smn.Done:
			// fmt.Println(_time, "上一级调用被取消了")
			utils.ErrorWithLableAndFormat("timer clock", "%v already done", _time)
			return
		default:
		}

		select {
		case <-smn.Chan:
			//TODO: 执行任务
			// utils.Debug("time clock", "%v active job worker", _time)
			//调起gorouting,把任务放入到worker中
			err := db.GetHandler().Update(func(tx *bolt.Tx) error {
				// secondBucket := dayBucket.Bucket([]byte(string(_time % 86400)))
				secondBucket, err := GetSecondBucketByTime(tx, _time)
				if err != nil {
					return nil
					// return err
				}
				for i := 1; i <= 10; i++ {
					priorityBucket := secondBucket.Bucket([]byte(string(i)))
					if priorityBucket == nil {
						continue
					}

					priorityBucket.ForEach(func(k, v []byte) error {
						job, err := GetJob(tx, v)
						// err := json.Unmarshal(v, job)
						if err != nil {
							utils.ErrorWithLableAndFormat("time clock", "json unmarshal error:%v id:%s", err, string(v))
							return nil
						}
						if job.Status != StatusFailed {
							wjpTasks <- workerJobTask{
								job: job,
								// bucket: priorityBucket,
							}
						}
						return nil
					})
				}

				// tx.Bucket("")
				return nil
			})
			if err != nil {
				utils.ErrorWithLableAndFormat("time clock", "db error:%v", err)
			}

			// ctx.Done()
			// done <- struct{}{}
			sm.Close(_time)
			return
		}
	}
}
