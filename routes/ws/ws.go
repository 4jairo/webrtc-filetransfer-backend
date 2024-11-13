package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	mongoclient "github.com/4jairo/webrtc-filetransfer-backendBackend/db"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

type WsRole int

const (
	WsRoleHost WsRole = iota
	WsRoleConn
)

func WsHandler(role WsRole) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)

		s := websocket.Server{Handler: websocket.Handler(func(c *websocket.Conn) {
			handleWs(c, vars["objId"], role)
		})}

		s.ServeHTTP(w, req)
	}
}

// if role is WsRoleHost, objId == filesId, else objId == signalingId
func handleWs(ws *websocket.Conn, objId string, role WsRole) {
	defer func() {
		ws.Close()
		if role == WsRoleHost {
			mongoclient.Mongo.DeleteFilesDoc(objId)
		} else {
			mongoclient.Mongo.DeleteSignalingDoc(objId)
		}
	}()

	for {
		var msg []byte
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			break
		}

		go func() {
			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				SendError(ws, fmt.Errorf("error decoding message: %v", err.Error()))
				return
			}

			msgData, err := message.GetDataType()
			if err != nil {
				SendError(ws, fmt.Errorf("error getting message kind: %v", err.Error()))
				return
			}

			if err := json.Unmarshal(message.Data, msgData); err != nil {
				SendError(ws, fmt.Errorf("error decoding message data: %v", err.Error()))
				return
			}

			var signalingDoc *string
			if role == WsRoleHost {
				signalingDoc = &message.SignalingId
			} else {
				signalingDoc = &objId
			}

			if _, err := msgData.Process(ws, signalingDoc); err != nil {
				SendError(ws, fmt.Errorf("error processing message: %v", err.Error()))
				return
			}
		}()
	}
}

func SendError(ws *websocket.Conn, err error) {
	msgError, _ := json.Marshal(MessageError{
		Msg: err.Error(),
	})

	msg := Message{
		Type: MsgError,
		Data: json.RawMessage(msgError),
	}
	msgBytes, _ := json.Marshal(msg)

	websocket.Message.Send(ws, string(msgBytes))
}
