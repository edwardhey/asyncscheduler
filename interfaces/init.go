package interfaces

type Resp struct {
	Code int `json:"code"`
	Data struct {
		AttemptInterval uint  `json:"attemptInterval,omitempty"` //重试间隔
		Priority        uint8 `json:"priority,omitempty"`        //权重
	} `json:"data,omitempty"`
	Err string `json:"error,omitempty"`
}

const (
	RespStatusErrorMissingAttributes = -9999999
)

type CallbackRequest struct {
	Code         int   `json:"code"` //code :0 成功  !=0: 失败
	AttemptTimes uint8 `json:"attemptTimes"`
}

type HttpResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error,omitempty"`
}
