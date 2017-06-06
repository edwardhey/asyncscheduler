package main

import (
	"golib/log"

	"edwardhey.com/asyncscheduler/http"
	"edwardhey.com/asyncscheduler/job"

	"github.com/spf13/viper"
)

// var l sync.Mutex

func main() {
	//wg := &sync.WaitGroup{}
	v := viper.New()
	v.SetConfigFile("./config.yaml")
	err := v.ReadInConfig()

	if err != nil {
		panic(err)
	}

	log.InitConsoleOutput(v.Get("log.level").(int))

	//wg.Add(1)

	go http.InitWithConfig(v)
	go job.InitWithViper(v)

	//wg.Wait()
	<-(chan struct{})(nil)
}
