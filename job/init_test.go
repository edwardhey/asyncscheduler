package job

import (
	"fmt"
	"golib/log"
	_ "net/http/pprof"
	"testing"
	"time"

	"edwardhey.com/asyncscheduler/utils"
	"github.com/boltdb/bolt"
)

func init() {
	log.InitConsoleOutput(log.LevelDebug)
	err := InitDB("/tmp/test.db")
	utils.ErrorWithLableAndFormat("init", "%v", err)
}

func TestBolt(t *testing.T) {
	db, err := bolt.Open("/tmp/a.db", 0700, nil)
	fmt.Println(err)
	go func() {
		log.Debug("1 entry")
		time.Sleep(3 * time.Second)
		db.View(func(tx *bolt.Tx) error {
			log.Debug("1 start")
			time.Sleep(10 * time.Second)
			log.Debug("1 end")
			return nil
		})
	}()

	go func() {
		log.Debug("2 entry")
		db.Batch(func(tx *bolt.Tx) error {
			log.Debug("2 start")
			time.Sleep(10 * time.Second)
			log.Debug("2 end")
			return nil
		})
	}()

	time.Sleep(100 * time.Second)
}

func _TestSturct2Bytes(t *testing.T) {
	AddJob(&Job{
		URL: "http://restapi.amap.com/v3/distance?origins=124.345235,43.145939&destination=116.367176,33.933020&output=json&key=e2ba190105c16695068c51dc85f36024&type=1",
		// IgnoreResponse: true,
		MaxAttempts:     1,
		AttemptInterval: 1,
		Callback:        "http://edwardhey.com",
	})

	AddJob(&Job{
		URL: "http://restapi.amap.com/v3/distance?origins=124.345235,43.145939&destination=116.367176,33.933020&output=json&key=e2ba190105c16695068c51dc85f36024&type=1",
		// IgnoreResponse: true,
		MaxAttempts:     1,
		AttemptInterval: 1,
		Callback:        "http://edwardhey.com",
	})

	AddJob(&Job{
		URL: "http://restapi.amap.com/v3/distance?origins=124.345235,43.145939&destination=116.367176,33.933020&output=json&key=e2ba190105c16695068c51dc85f36024&type=1",
		// IgnoreResponse: true,
		MaxAttempts:     1,
		AttemptInterval: 1,
		Callback:        "http://edwardhey.com",
	})
	// utils.ErrorWithLableAndFormat("test", "%v", err)
	// go InitGoRouting()
	// http.ListenAndServe("localhost:6060", nil)
	time.Sleep(1800 * time.Second)
}
