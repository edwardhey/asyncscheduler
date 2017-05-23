package http

import (
	"golib/link/httplink"
	"net/http"
	"time"

	"io/ioutil"

	"encoding/json"

	"edwardhey.com/asyncscheduler/interfaces"
	"edwardhey.com/asyncscheduler/job"
	"github.com/labstack/echo"
	"github.com/spf13/viper"
)

func InitWithConfig(v *viper.Viper) {

	httplink.SetDefaultSetting(httplink.Settings{
		UserAgent:        "AsyncschedulerServer",
		ConnectTimeout:   time.Duration(v.Get("job.connect_timeout").(int)) * time.Second,
		ReadWriteTimeout: time.Duration(v.Get("job.read_write_timeout").(int)) * time.Second,
		Gzip:             true,
		DumpBody:         true,
	})

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return nil
	})
	e.POST("/job/set", SetJob)
	e.GET("/job/get", GetJob)
	e.GET("job/del", DelJob)
	e.GET("/job/dumpall", DumpAllJobs)

	addr := v.Get("http.address").(string)
	if addr == "" {
		panic("http addresss not set")
	}
	e.Start(addr)
}

// func ReturnJsonResponse()

func DumpAllJobs(c echo.Context) error {
	// retu
	jobs, _ := job.GetJobsWithoutTx()
	return c.JSON(http.StatusAccepted, jobs)
	// return nil
}

func GetJob(c echo.Context) error {
	id := c.FormValue("ID")
	if id == "" {
		return c.JSON(http.StatusBadRequest, &interfaces.HttpResponse{
			Code:  http.StatusBadRequest,
			Error: "ID为空",
		})
	}

	j := job.GetJobWithoutTx(id)
	if j != nil {
		// job.GetJob()
		return c.JSON(http.StatusAccepted, j)
	}
	return c.JSON(http.StatusNotFound, &interfaces.HttpResponse{
		Code:  http.StatusNotFound,
		Error: "job not found",
	})
}

func DelJob(c echo.Context) error {
	id := c.FormValue("ID")
	if id == "" {
		return c.JSON(http.StatusBadRequest, &interfaces.HttpResponse{
			Code:  http.StatusBadRequest,
			Error: "ID为空",
		})
	}
	job.DelJobWithoutTx(id)
	return c.JSON(http.StatusAccepted, &interfaces.HttpResponse{
		Code: http.StatusAccepted,
	})
}

func SetJob(c echo.Context) error {

	s, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &interfaces.HttpResponse{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		})
	}
	// defer close(s)
	j := &job.Job{}
	if err := json.Unmarshal(s, j); err != nil {
		return c.JSON(http.StatusBadRequest, &interfaces.HttpResponse{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		})
	}
	// fmt.Println(job)
	// return nil
	// j.IgnoreResponse = true
	// j.MaxAttempts = 10
	// j.AttemptInterval = 1
	err = job.AddJob(j)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, &interfaces.HttpResponse{
			Code:  http.StatusServiceUnavailable,
			Error: err.Error(),
		})
	}
	return c.JSON(http.StatusAccepted, &interfaces.HttpResponse{})
	// j.Payload = payload
	// job.AddJob()

	// return nil
}
