package job

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"edwardhey.com/asyncscheduler/interfaces"
	"edwardhey.com/asyncscheduler/utils"

	"github.com/boltdb/bolt"
)

//MaxJobNums 最大任务数
var MaxJobNums int
var TotalJobNums int
var RunJobNums int

var jobBucketName []byte = []byte("jobs")

type Method string

const (
	// jobDaysBucketName string = "jobDays"    //有任务的日期，用来遍历数据
	// finishedBucketName string = "finishedJobs"  //已完成，过期的将会被删除
	// MaxJobNums          int    = 65535           //最大任务数
	jobDayBucketName  string = "job_%s"       //任务详细数据
	jobDatetimeFormat string = "2006-01-02"   //任务日期格
	MethodPost        Method = Method("POST") //post
	MethodGet         Method = Method("Get")  //get
	StatusPedding            = 0              //就绪中
	StatusRetrying           = -10            //重试中
	StatusRetryFailed        = -20            //重试失败，等待再次重试
	StatusFinished           = 100            //已完成
	StatusFailed             = -100           //已失败
)

//Job 任务
type Job struct {
	ID                string     `json:"id"`                        //Job的ID
	URL               string     `json:"url"`                       //请求的Url地址
	TTL               uint32     `json:"ttl"`                       //过期时间
	TTR               uint32     `json:"ttr"`                       //执行时间
	Payload           url.Values `json:"payload"`                   //内容
	Method            Method     `json:"method"`                    //方法
	Priority          uint8      `json:"priority"`                  //权重
	MaxAttempts       uint8      `json:"max_attempts,string"`       //最大尝试次数
	AttemptTimes      uint8      `json:"attempt_times,string"`      //重试次数
	AttemptInterval   uint       `json:"attempt_interval,string"`   //重试间隔
	ConnectTimeout    uint8      `json:"connect_timeout,string"`    //链接超时
	ReadWriteTimeout  uint8      `json:"read_write_timeout,string"` //读写超时
	Callback          string     `json:"callback"`                  //回调地址
	IsCallbackSuccess bool       `json:"is_callback_success"`       //是否通知成功
	Status            int8       `json:"status,string"`             //状态
	IgnoreResponse    bool       `json:"ignore_response"`           //是否忽略返回内容，只要判断状态为200，就是视为成功
	Ctime             uint32     `json:"ctime"`                     //创建时间
	Mtime             uint32     `json:"mtime"`                     //更改时间
}

func (j *Job) GetJobKey() string {
	return string(j.ID)
	// return utils.Int64ToBytes(uint64(j.ID))
}

func (j *Job) Json() ([]byte, error) {
	jb, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return jb, nil
}

func (j *Job) Failed(err error, resp *interfaces.Resp) {
	// isSuccess := false
	utils.ErrorWithLableAndFormat("Consume Task", "job:%v http error:%v", j.ID, err)
	now := getNowTimestamp()
	j.Status = StatusRetryFailed
	for {
		if j.CheckValidate() {
			if resp != nil {
				if resp.Data.AttemptInterval > 0 && resp.Data.AttemptInterval != j.AttemptInterval {
					j.AttemptInterval = resp.Data.AttemptInterval
				}

				j.TTR = uint32(j.AttemptInterval) + now

				if resp.Data.Priority > 0 && resp.Data.Priority != j.Priority {
					utils.InfoWithLableAndFormat("Consume Task", "job:%v priority chaned", j.ID)

					//新建一个任务
					j.Priority = resp.Data.Priority
				}

			} else {
				j.TTR = uint32(j.AttemptInterval) + now
				// break
				// fmt.Println(j.TTR, getNowTimestamp(), now)
			}
			break
			// return nil
		}
		j.Status = StatusFailed
		break
	}
	// utils.DebugWithLableAndFormat("Consume Task", "job:%+v", j)
}

func (j *Job) Success() {
	j.Status = StatusFinished
}

func (j *Job) CheckValidate() bool {
	// fmt.Println(j.MaxAttempts, j.AttemptTimes, j.TTL, getNowTimestamp())
	if j.MaxAttempts <= j.AttemptTimes {
		return false
	} else if j.TTL >= 0 && j.TTL < getNowTimestamp() {
		return false
	}
	return true
}

func GetJob(tx *bolt.Tx, jobId []byte) (*Job, error) {
	bucket := tx.Bucket(jobBucketName)
	if bucket == nil {
		return nil, fmt.Errorf("job bucket not found")
	}
	data := bucket.Get(jobId)
	if data == nil {
		return nil, fmt.Errorf("job not found")
	}

	job := &Job{}
	if err := json.Unmarshal(data, job); err != nil {
		return nil, err
	}
	return job, nil
}

func DelJobWithoutTx(key string) bool {
	db.GetHandler().Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(jobBucketName)
		b.Delete([]byte(key))
		return nil
	})
	return true
}

func GetJobWithoutTx(key string) *Job {
	//jobs := make(map[string]*Job, 100)
	job := &Job{}
	err := db.GetHandler().View(func(tx *bolt.Tx) error {
		b := tx.Bucket(jobBucketName)
		v := b.Get([]byte(key))
		// fmt.Println(key, v)
		return json.Unmarshal(v, job)
		// return nil
	})

	if err != nil {
		return nil
	}
	return job

}

func GetJobsWithoutTx() (map[string]*Job, error) {
	jobs := make(map[string]*Job, 100)
	db.GetHandler().View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(jobBucketName)
		if bucket == nil {
			return fmt.Errorf("job bucket not found")
		}

		bucket.ForEach(func(name, val []byte) error {
			// fmt.Println(string(name), string(val))
			job := &Job{}
			json.Unmarshal(val, job)
			jobs[string(name)] = job
			return nil
		})
		return nil
	})
	return jobs, nil
}

func (j *Job) Save(tx *bolt.Tx) error {
	utils.DebugWithLableAndFormat("set job", "%+v", j)
	// fmt.Println(j.TTR, j.Priority)

	priorityBucket, err := CreatePriorityBucketByTimeAndPriority(tx, j.TTR, j.Priority)
	if err != nil {
		return err
	}

	// fmt.Println([]byte(j.ID), string([]byte(j.ID)))
	jb, err := j.Json()
	if err != nil {
		return err
	}

	// fmt.Println("getBucket", jobBucketName)
	jobsBuckets := tx.Bucket(jobBucketName)
	if jobsBuckets == nil {
		return err
	}
	// fmt.Println(j.GetJobKey(), string(jb))
	if err := jobsBuckets.Put([]byte(j.GetJobKey()), jb); err != nil {
		return err
	}

	//索引更新
	if err := priorityBucket.Put([]byte(j.GetJobKey()), []byte(j.ID)); err != nil {
		return err
	}

	return nil
}

func DeleteBucketByDay(tx *bolt.Tx, _t uint32) error {
	// now := uint32(time.Now().Unix())
	// b := tx.Bucket([]byte(jobDaysBucketName))
	t := time.Unix(int64(_t), 0)
	datetime := t.Format(jobDatetimeFormat)
	return tx.DeleteBucket([]byte(fmt.Sprintf(jobDayBucketName, datetime)))
	// return
}

func GetDayBucketByTime(tx *bolt.Tx, _t uint32) (*bolt.Bucket, error) {
	// now := uint32(time.Now().Unix())
	// b := tx.Bucket([]byte(jobDaysBucketName))
	t := time.Unix(int64(_t), 0)
	datetime := t.Format(jobDatetimeFormat)
	// fmt.Println(t, _t, _t%86400)
	// if string(b.Get([]byte(datetime))) == "" {
	// 	return nil, fmt.Errorf("bucket not found")
	// }

	dayBucket := tx.Bucket([]byte(fmt.Sprintf(jobDayBucketName, datetime)))
	if dayBucket == nil {
		return nil, fmt.Errorf("day bucket not found")
	}
	return dayBucket, nil
}

func CreateDayBucketByTime(tx *bolt.Tx, _t uint32) (*bolt.Bucket, error) {
	// b := tx.Bucket([]byte(jobDaysBucketName))
	t := time.Unix(int64(_t), 0)
	datetime := t.Format(jobDatetimeFormat)
	// fmt.Println(t, _t, _t%86400)
	// if err := b.Put([]byte(datetime), []byte("1")); err != nil {
	// 	return nil, err
	// }

	// ...read or write...
	return tx.CreateBucketIfNotExists([]byte(fmt.Sprintf(jobDayBucketName, datetime)))
}

func GetSecondBucketByTime(tx *bolt.Tx, _t uint32) (*bolt.Bucket, error) {
	var dayBucket *bolt.Bucket
	var err error
	if dayBucket, err = GetDayBucketByTime(tx, _t); err != nil {
		return nil, err
	}
	__t := _t % 86400
	secondBucket := dayBucket.Bucket([]byte(string(__t)))
	if secondBucket == nil {
		return nil, fmt.Errorf("second bucket:%v not found", __t)
	}
	return secondBucket, nil
}

func CreateSecondBucketByTime(tx *bolt.Tx, _t uint32) (*bolt.Bucket, error) {
	var dayBucket *bolt.Bucket
	var err error
	if dayBucket, err = CreateDayBucketByTime(tx, _t); err != nil {
		return nil, err
	}
	// fmt.Println(_t % 86400)
	__t := _t % 86400
	return dayBucket.CreateBucketIfNotExists([]byte(string(__t)))
}

func GetPriorityBucketByTimeAndPriority(tx *bolt.Tx, _t uint32, _p uint8) (*bolt.Bucket, error) {
	priorityBucket, err := CreatePriorityBucketByTimeAndPriority(tx, _t, _p)
	if priorityBucket == nil {
		return nil, fmt.Errorf("priority bucket not found")
	} else if err != nil {
		return nil, err
	}

	return priorityBucket, nil
}

func CreatePriorityBucketByTimeAndPriority(tx *bolt.Tx, _t uint32, _p uint8) (*bolt.Bucket, error) {
	var secondBucket *bolt.Bucket
	var err error
	if secondBucket, err = CreateSecondBucketByTime(tx, _t); err != nil {
		return nil, err
	}
	// for i := 1; i <= 10; i++ {
	return secondBucket.CreateBucketIfNotExists([]byte(string(_p)))
}

//AddJob 增加任务
func AddJob(j *Job) error {

	if j.URL == "" {
		return fmt.Errorf("url cannot empty")
	}
	now := getNowTimestamp()
	if j.TTL <= now || j.TTL == 0 {
		// return fmt.Errorf("过期时间比当前时间要少")
		j.TTL = now + 300 //测试，5分钟后过期
	}
	if j.TTR <= now {
		j.TTR = now + 1
	}

	if j.ID == "" { //生成jobid
		id, err := idGenr.Next()
		if err != nil {
			return err
		}
		// fmt.Println(id)
		j.ID = fmt.Sprintf("%v", id)
	}
	if j.Priority == 0 {
		j.Priority = 1
	} else if j.Priority > 10 {
		return fmt.Errorf("job priority should >0 and <=10")
	}

	if j.MaxAttempts == 0 {
		j.MaxAttempts = 1
	}

	if j.AttemptInterval == 0 {
		j.AttemptInterval = 1
	}

	j.Ctime = getNowTimestamp()
	j.Mtime = j.Ctime
	j.Status = StatusPedding

	return db.GetHandler().Update(func(tx *bolt.Tx) error {
		j.Save(tx)
		// fmt.Println("add job end")
		return nil
	})
}
