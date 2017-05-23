package job

import (
	"time"

	"encoding/json"

	"edwardhey.com/asyncscheduler/utils"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
)

const (
	Day1  = 86400
	Days3 = 3 * 86400
)

var min []byte = []byte("1990-01-01T00:00:00Z")

func flushJobsWithRetryingToNewBuckets() {
	utils.InfoWithLable("boot db", "flush jobs with retrying status to new bucket")

	//刷新过去的数据到
	db.GetHandler().Update(func(tx *bolt.Tx) error {

		b := tx.Bucket(jobBucketName)
		c := b.Cursor()

		now := getNowTimestamp()
		timeout := now - Day1
		for k, v := c.First(); k != nil; k, v = c.Next() {
			job := &Job{}
			err := json.Unmarshal(v, job)
			if err != nil {
				utils.ErrorWithLable("boot db", "json decode error:%s", err)
			}

			if job.Ctime < timeout {
				break
			} else if job.CheckValidate() && job.TTR <= now {
				job.TTR = getNowTimestamp() + 10 //10秒后执行
				job.Save(tx)
			}
			// fmt.Println(string(k), now, job.TTR)

			// continue
			// AddJob(job)
			/*
				if job.Ctime >= timeout {
					fmt.Println(job.Ctime, timeout)
					break
				} else if job.CheckValidate() == false && (job.Status == StatusRetrying || job.Status == StatusPedding) {
					job.TTR = getNowTimestamp() + 10 //10秒后执行
					AddJob(job)
				}*/
		}
		// fmt.Println("done")
		return nil
	})
}

func rebuildDb() {
	_cron := cron.New()
	_cron.AddFunc("0 0 0 * *", func() { //0点执行
		db.GetHandler().Update(func(tx *bolt.Tx) error {
			utils.InfoWithLable("rebuild db", nil)
			// Assume our events bucket exists and has RFC3339 encoded time keys.
			b := tx.Bucket(jobBucketName)
			c := b.Cursor()

			//数据保留三天
			now := getNowTimestamp()
			timeout := now - Days3
			for k, v := c.First(); k != nil; k, v = c.Next() {
				job := &Job{}
				err := json.Unmarshal(v, job)
				if err != nil {
					utils.ErrorWithLable("rebuild db", "json decode error:%s", err)
				}
				if job.Ctime >= timeout {
					break
				} else if job.CheckValidate() == false && job.Ctime < timeout {
					utils.InfoWithLableAndFormat("rebuild db", "delete job id:%s", job.ID)
					b.Delete(k)
				}
			}

			// tx.ForEach(func())

			return nil
		})
	})
	_cron.Start()
}
func GetDBHandler() *bolt.DB {
	return db.GetHandler()
}

func removeOldBucketIndex() {
	// go func() {
	//60秒维护一下索引数据库
	internal := 300
	ticker := time.NewTicker(time.Duration(internal) * time.Second)
	// bucket := db.GetHandler().
	for {
		time := <-ticker.C
		timeInt := uint32(time.Unix())
		timeMod := timeInt % 86400
		utils.InfoWithLable("remove old bucket")
		db.GetHandler().Update(func(tx *bolt.Tx) error {
			if 10 > timeMod { //把整个bucket删除
				if err := DeleteBucketByDay(tx, timeInt-uint32(internal)); err != nil {
					utils.InfoWithLableAndFormat("remove old bucket", "delete yesterday bucket error:%v", err)
				}

			}

			bucket, err := GetDayBucketByTime(tx, timeInt)
			if err != nil {
				return nil
			}
			// fmt.Println("aaa", timeInt, bucket, err)
			// bucket := tx.Bucket(jobBucketName)
			for i := timeMod - uint32(2*internal+1); i < timeMod-uint32(internal); i++ {
				bucket.DeleteBucket([]byte(string(i)))
				// if err := bucket.DeleteBucket([]byte(string(i))); err != nil {
				// utils.DebugWithLableAndFormat("remove old bucket", "delete %d bucket error:%v", i, err)
				// }
			}

			return nil
		})
		// sm.SendSignal()
	}
	// }()
	// return
}
