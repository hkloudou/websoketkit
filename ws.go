package websoketkit

import (
	"encoding/json"
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
	m.Broadcasts = make(chan Broadcast, 4096)
	m.Functions = make(chan FunctionData, 0)
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
			//log.Println("SessionID:", obj.(*ConnectData).SessionID, "Closed")
		} else if obj.(*ConnectData).NeedClose == true {
			//log.Println("SessionID:", obj.(*ConnectData).SessionID, "NeedClose")
		}
		ws.Close()
		m.Connects.Delete(ws)
	}()
	guid := uuid.NewV4()
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
			} else if html := string(message); 1 == 2 {

			} else if html == "ping" {
				m.WriteMsgByID(obj.(*ConnectData).SessionID, "system", "pong")
			} else if !gjson.Valid(html) {
				// 错误数据
				log.Println("un valid data", html)
			} else if gjson.Parse(html).IsArray() {
				//log.Println("gjson.Parse(html).IsArray()", html)
				for _, item := range gjson.Parse(html).Array() {
					m.deelData(obj.(*ConnectData), item.String())
				}
			} else {
				//log.Println("m.deelData(obj.(*ConnectData), html)", html)
				m.deelData(obj.(*ConnectData), html)
			}
		}
	}
}

func (m *WebsocketHandler) deelData(connData *ConnectData, html string) {
	if html == "ping" {
		m.WriteMsgByID(connData.SessionID, "system", "pong")
	} else if gjson.Get(html, "action").String() == "sub" && gjson.Get(html, "channel").Exists() {
		data := &SubscriptionData{
			SessionID: connData.SessionID,
			Channel:   gjson.Get(html, "channel").String(),
			Data:      "",
		}
		connData.Subscriptions.Store(gjson.Get(html, "channel").String(), data)
	} else if (gjson.Get(html, "action").String() == "fun" || gjson.Get(html, "action").String() == "func") && gjson.Get(html, "funcname").Exists() {
		data := FunctionData{
			SessionID: connData.SessionID,
			FuncName:  gjson.Get(html, "funcname").String(),
			Parame:    nil,
		}

		if gjson.Get(html, "parame").IsObject() {
			v := make(map[string]interface{}, 0)
			if err := json.Unmarshal([]byte(gjson.Get(html, "parame").String()), &v); err == nil {
				data.Parame = v
			}
		}
		if FireFunc(data) {

		} else {
			go func() {
				select {
				case m.Functions <- data:
					//log.Println("deel func:", string(message))
				case <-time.After(1 * time.Second):
					log.Println("un deel func:", html, "please use<-m.Functions to recive function")
				}
			}()
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
			} else if msg.SessionID == "" && msg.ChannelName == "" {
				//no seessionid and no channel name
				return true
			} else {
				//log.Println("write:", msg)
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
