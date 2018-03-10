package websoketkit

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
)

//Broadcast Broadcast
type Broadcast struct {
	SessionID              string      `json:"-"`
	ChannelName            string      `json:"c"`
	Msg                    interface{} `json:"m"`
	CreatAt                int64       `json:"t"`
	ISMessageForSeessionID bool        `json:"-"`
}

//WebsocketHandler WebsocketHandler
type WebsocketHandler struct {
	Inited     bool
	Upgrader   websocket.Upgrader
	Connects   sync.Map
	Broadcasts chan Broadcast
	Functions  chan FunctionData
}

//Init remenber
func (m *WebsocketHandler) Init() {
	if m.Inited {
		return
	}
	m.Inited = true
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	m.Upgrader = upgrader
	m.Broadcasts = make(chan Broadcast, 512)
	m.Functions = make(chan FunctionData)
	go m.HandleWebsocketMessages()
}

//ServeHTTP Register websocket
func (m *WebsocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Init()
	ws, err := m.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	//This defer to clear timeout connect
	defer func() {
		if obj, ok := m.Connects.Load(ws); !ok {
			log.Println("SessionID:", obj.(*ConnectData).SessionID, "Closed")
		} else if obj.(*ConnectData).NeedClose == true {
			log.Println("SessionID:", obj.(*ConnectData).SessionID, "NeedClose")
		}
		ws.Close()
		m.Connects.Delete(ws)
	}()
	guid, _ := uuid.NewV4()
	data := &ConnectData{OnlineAt: time.Now()}
	data.SessionID = guid.String()
	m.Connects.Store(ws, data)

	//Read message
	for {
		if obj, ok := m.Connects.Load(ws); !ok {
			return //No Session on Memory,deleted,jump to defer
		} else if obj.(*ConnectData).NeedClose {
			return //NeedClose,jump to defer to close
		} else {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				obj.(*ConnectData).NeedClose = true
				return
			} else if mt != 1 {
				//log.Println("mt:" + string(mt))
			} else if mt == 1 && string(message) == "ping" {
				m.WriteMsgByID(obj.(*ConnectData).SessionID, "system", "pong")
			} else if mt == 1 && !gjson.Valid(string(message)) {
				//log.Println("read:", "err format")
			} else if mt == 1 && gjson.Get(string(message), "action").String() == "sub" && gjson.Get(string(message), "channel").Exists() {
				//log.Println("recive:", string(message))
				data := &SubscriptionData{
					SessionID: obj.(*ConnectData).SessionID,
					Channel:   gjson.Get(string(message), "channel").String(),
					Data:      "",
				}
				//log.Println("data:", data)
				//log.Println("gjson.Get(string(message), \"channel\").String()", gjson.Get(string(message), "channel").String())
				obj.(*ConnectData).Subscriptions.Store(gjson.Get(string(message), "channel").String(), data)
			} else if mt == 1 && gjson.Get(string(message), "action").String() == "fun" && gjson.Get(string(message), "funcname").Exists() {
				data := FunctionData{
					SessionID: obj.(*ConnectData).SessionID,
					FuncName:  gjson.Get(string(message), "funcname").String(),
					Parame:    nil,
				}
				//log.Println("func", data)
				go func() {
					select {
					case <-time.After(200 * time.Millisecond):
						log.Println("un deel func:", string(message), "please use<-m.Functions to recive function")
					case m.Functions <- data:
						log.Println("deel func:", string(message))
					}
				}()
			}
		}
	}
}

//WriteMsgByID WriteMsg
func (m *WebsocketHandler) WriteMsgByID(sessionID string, channelName string, msg interface{}) {
	m.Init()
	go func() {
		m.Broadcasts <- Broadcast{
			Msg:                    msg,
			SessionID:              sessionID,
			ChannelName:            channelName,
			CreatAt:                time.Now().UnixNano(),
			ISMessageForSeessionID: true,
		}
	}()
}

//WriteMsgByChannelName WriteMsg
func (m *WebsocketHandler) WriteMsgByChannelName(channelName string, msg interface{}) {
	m.Init()
	go func() {
		m.Broadcasts <- Broadcast{
			Msg:                    msg,
			ChannelName:            channelName,
			CreatAt:                time.Now().UnixNano(),
			ISMessageForSeessionID: false,
		}
	}()
}

//HandleWebsocketMessages HandleWebsocketMessages
func (m *WebsocketHandler) HandleWebsocketMessages() {
	//log.Println(" HandleWebsocketMessages")
	m.Init()
	for {
		msg := <-m.Broadcasts
		m.Connects.Range(func(con, data interface{}) bool {
			if data.(*ConnectData).NeedClose {
				//log.Println("HandleWebsocketMessages NeedClose")
				return true //Continue Range
			} else if msg.ISMessageForSeessionID && msg.SessionID != "" && data.(*ConnectData).SessionID != msg.SessionID {
				// if sessionid way and sessionid Exists,but diffrent.
				//log.Println("sessionid: ", msg.SessionID, " Exists,but diffrent.")
				return true //Continue Range
			} else if _, ok := data.(*ConnectData).Subscriptions.Load(msg.ChannelName); !ok && msg.ChannelName != "" && !msg.ISMessageForSeessionID {
				// if not sessionid way ,ChannelName Exists,but this connect not subcription.
				//log.Println("ChannelName: ", msg.ChannelName, " Exists,but diffrent.")
				return true //Continue Range
			} else if msg.SessionID == "" || msg.ChannelName == "" {
				//no seessionid and no channel name
				return true
			} else {
				//sessionid or ChannelName Exists
				//log.Println("sessionid or ChannelName Exists")
				err := con.(*websocket.Conn).WriteJSON(msg)
				if err != nil {
					con.(*websocket.Conn).Close()
					data.(*ConnectData).NeedClose = true
				} else {
					data.(*ConnectData).LastSendAt = time.Now()
				}
				return true //Continu range to send
			}
		})
	}
}
