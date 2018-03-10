package websoketkit

import (
	"sync"
	"time"
)

//ConnectData 链接数据
type ConnectData struct {
	SessionID     string `json:"sessionid"`
	NeedClose     bool   `json:"-"`
	Subscriptions sync.Map
	Functions     chan FunctionData
	OnlineAt      time.Time `json:"online_at"`
	LastSendAt    time.Time `json:"lastsend_at"`
}
