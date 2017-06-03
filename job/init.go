package job

import (
	"golib/dba"
	"time"

	"github.com/boltdb/bolt"

	"edwardhey.com/asyncscheduler/utils"
	"github.com/spf13/viper"
)

var db *dba.DBABolt
var idGenr *utils.SnowFlake

var wjp *workerJobPool
var wjpTasks chan workerJobTask

type contextKey string

func init() {
	idGenr, _ = utils.NewSnowFlake(utils.BusinessJob)
}

func InitTaskBuffer(v *viper.Viper) {
	wjpTasks = make(chan workerJobTask, v.Get("job.task_chan_buffer_size").(int))
	wjp = &workerJobPool{
		tasks:    wjpTasks,
		poolSize: v.Get("job.pool_size").(int),
	}
	go wjp.Run()
}

func getNowTimestamp() uint32 {
	return uint32(time.Now().Unix())
}

//----------------------------db---------------------------------------

//InitDB 初始化DB
func InitDB(path string) error {
	db = dba.NewBolt()
	err := db.Open(path, 0700)
	if err != nil {
		return err
	}
	return db.GetHandler().Batch(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(jobBucketName)
		if err != nil {
			return err
		}
		return nil
	})
}

//---------------------------------singnalMap-------------------------------
var sm *signalMap

func InitWithViper(v *viper.Viper) {

	dbfile := v.Get("job.db").(string)
	if dbfile == "" {
		panic("db file cannot empty")
	}
	InitDB(dbfile)

	MaxJobNums = v.Get("job.maxnums").(int)
	if MaxJobNums == 0 {
		MaxJobNums = 65535
	}

	nums := v.Get("job.poolnums").(int)
	if nums == 0 {
		nums = 10
	}

	InitTaskBuffer(v)

	sm = newSignalMap(nums)
	//不断的启动gorouting
	// go func(t uint32) {
	// 	for {
	// 		// fmt.Println("----------------------")
	// 		timeNow := getNowTimestamp()

	// 		for i := timeNow; i < timeNow+t; i++ {
	// 			go createTimerWorker(timeNow, i, t)
	// 		}
	// 		time.Sleep(time.Duration(t) * time.Second)
	// 	}
	// }(uint32(nums))

	//到时间点了以后往chan发信号激活
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			//起gorouting把任务从bucket放入到pool中
			_t := <-ticker.C
			go func(_time uint32) {
				// fmt.Println(_time)
				err := db.GetHandler().Update(func(tx *bolt.Tx) error {
					// fmt.Println(_time % 86400)
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
							// fmt.Println("aaaaaaaaaaaa", job.ID, job)
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
					return nil
				})
				if err != nil {
					utils.ErrorWithLableAndFormat("time clock", "db error:%v", err)
				}
				// tx.Bucket("")
				// return nil
			}(uint32(_t.Unix()))
			// if err != nil {
			// 	utils.ErrorWithLableAndFormat("time clock", "db error:%v", err)
			// }
			// sm.SendSignal(uint32(time.Unix()))

		}
	}()

	//删除bucket索引
	go removeOldBucketIndex()
	//删除过期的数据
	go rebuildDb()
	go flushJobsWithRetryingToNewBuckets()

}
