# Asyncscheduler 定时任务调度器
========
这是一个带有权重的定时任务调度器，只支持http的方式调度


## 功能
1. 定时执行
2. 执行结果回调通知
3. 任务数据保留3天
4. 任务权重设置，1~10，10为最高权重
5. 设置指定的jobID，若不设置会自动生成jobID，若指定则会覆盖相同ID的任务


## 使用

```bash
#!/bin/bash
for i in $(seq 1 1000)
do
	echo $i
	## SetJob
	curl -X "POST" "http://localhost:1984/job/set?ID=111&URL=http:%2F%2Fchargerlink.com&maxAttempts=5" \
	     -H "Content-Type: application/json; charset=utf-8" \
	     -d $'{
	  "id": "'$i'",
	  "payload": {
	    "aa": "bb"
	  },
	  "max_attempts": "10",
	  "url": "http://charger.com",
	  "attempt_interval": "1"
	}'

done
```


## Job的结构体
```golang
//Job 任务
type Job struct {
	ID                string      `json:"id"`                        //Job的ID
	URL               string      `json:"url"`                       //请求的Url地址
	TTL               uint32      `json:"ttl"`                       //过期时间
	TTR               uint32      `json:"ttr"`                       //执行时间
	Payload           interface{} `json:"payload"`                   //内容
	Priority          uint8       `json:"priority"`                  //权重
	MaxAttempts       uint8       `json:"max_attempts,string"`       //最大尝试次数
	AttemptTimes      uint8       `json:"attempt_times,string"`      //重试次数
	AttemptInterval   uint        `json:"attempt_interval,string"`   //重试间隔
	ConnectTimeout    uint8       `json:"connect_timeout,string"`    //链接超时
	ReadWriteTimeout  uint8       `json:"read_write_timeout,string"` //读写超时
	Callback          string      `json:"callback"`                  //回调地址
	IsCallbackSuccess bool        `json:"is_callback_success"`       //是否通知成功
	Status            int8        `json:"status,string"`             //状态
	IgnoreResponse    bool        `json:"ignore_response"`           //是否忽略返回内容，只要判断状态为200，就是视为成功
	Ctime             uint32      `json:"ctime"`                     //创建时间
	Mtime             uint32      `json:"mtime"`                     //更改时间
}
```


