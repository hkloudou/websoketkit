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

//WebsocketMessage WebsocketMessage
type WebsocketMessage struct {
	Message interface{} `json:"msg"`
}

//Broadcast Broadcast
type Broadcast struct {
	SessionID string
	Msg       WebsocketMessage
}

//WebsocketHandler WebsocketHandler
type WebsocketHandler struct {
	Inited     bool
	Upgrader   websocket.Upgrader
	Connects   sync.Map
	Broadcasts chan Broadcast
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
	m.Broadcasts = make(chan Broadcast, 1)
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
			log.Println("Closed connect")
		} else if obj.(*ConnectData).NeedClose == true {
			log.Println("SessionID:", obj.(*ConnectData).SessionID, "Closed")
		}
		ws.Close()
		m.Connects.Delete(ws)
	}()
	guid, _ := uuid.NewV4()
	data := &ConnectData{OnlineAt: time.Now()}
	data.SessionID = guid.String()
	m.Connects.Store(ws, data)

	/*
		go func() {

		}()*/

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
				log.Println("mt:" + string(mt))
			} else if mt == 1 && !gjson.Valid(string(message)) {
				log.Println("read:", "err format")
			} else if mt == 1 && gjson.Get(string(message), "a").String() == "live" {

			}
		}
	}

	/*
		//Write Message
		for {
			if obj, ok := m.Connects.Load(ws); !ok {
				return //No Session on Memory,deleted,jump to defer
			} else if obj.(*ConnectData).NeedClose {
				return //NeedClose,jump to defer to close
			} else {
				msg := Broadcast{Msg: WebsocketMessage{Message: ""}}

				if obj.(*ConnectData).SessionID != "" {
					//获取到imei 之前别发送消息吧
					// = obj.Imei
					m.Broadcasts <- msg
				}
			}

			time.Sleep(time.Second * 4)
		}
	*/
}

//WriteMsg WriteMsg
func (m *WebsocketHandler) WriteMsg(sessionID string) {
	msg := Broadcast{Msg: WebsocketMessage{Message: ""}, SessionID: sessionID}
	m.Broadcasts <- msg
}

//HandleWebsocketMessages 广播发送至页面
func (m *WebsocketHandler) HandleWebsocketMessages() {
	go func() {
		for {
			msg := <-m.Broadcasts
			m.Connects.Range(func(con, data interface{}) bool {
				if data.(*ConnectData).SessionID != msg.SessionID || msg.SessionID == "" {
					return true
				} else if data.(*ConnectData).NeedClose {
					return true
				} else {
					err := con.(*websocket.Conn).WriteJSON(msg.Msg)
					if err != nil {
						con.(*websocket.Conn).Close()
						data.(*ConnectData).NeedClose = true
					} else {
						data.(*ConnectData).LastSendAt = time.Now()
					}
				}
				return false
				//con.(*websocket.Conn).

			})
			/*
				for con, data := range Connects {

					if data.Imei != msg.Imei || msg.Imei == "" {
						continue
					}

					if data.NeedClose {
						continue
					}
					err := con.WriteJSON(msg.Msg)
					if err != nil {
						con.Close()
						data.NeedClose = true
					} else {
						data.LastSendAt = time.Now()
						logDevice(data)
					}
				}
			*/
		}
	}()
}
