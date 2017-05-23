package main

import (
	"golib/log"

	"edwardhey.com/asyncscheduler/http"
	"edwardhey.com/asyncscheduler/job"

	"sync"

	"github.com/spf13/viper"
)

var l sync.Mutex

func main() {
	wg := &sync.WaitGroup{}
	v := viper.New()
	v.SetConfigFile("./config.yaml")
	err := v.ReadInConfig()
	// viper.SetConfigFile("./config.yaml")
	// err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	log.InitConsoleOutput(log.LevelDebug)
	// job.InitDB()

	wg.Add(1)

	// viper.Get("http.port"), err)
	go http.InitWithConfig(v)
	go job.InitWithViper(v)

	// echo.

	wg.Wait()
}
