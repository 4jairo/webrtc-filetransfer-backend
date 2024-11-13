package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	mongoclient "github.com/4jairo/webrtcFiletransfer/db"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/net/websocket"
)

type MessageProcessor interface {
	Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error)
}

type MessageType int

const (
	MsgListenOffersHost MessageType = iota
	MsgListenOffersConn
	MsgOfferIceCandidate
	MsgAnswerIceCandidate
	MsgNewAnswer
	MsgNewOffer
	MsgError
)

type Message struct {
	Type        MessageType     `json:"type" validate:"required"`
	SignalingId string          `json:"signalingId,omitempty"`
	Data        json.RawMessage `json:"data"`
}

func (m *Message) GetDataType() (MessageProcessor, error) {
	switch m.Type {
	case MsgListenOffersHost:
		var msg ListenOffersHost
		return &msg, nil

	case MsgListenOffersConn:
		var msg ListenOffersConn
		return &msg, nil

	case MsgAnswerIceCandidate:
		var msg IceAnswerCandidate
		return &msg, nil

	case MsgOfferIceCandidate:
		var msg IceOfferCandidate
		return &msg, nil

	case MsgNewAnswer:
		var msg NewAnswer
		return &msg, nil

	case MsgNewOffer:
		var msg NewOffer
		return &msg, nil

	default:
		return nil, fmt.Errorf("unknown message type %v", m.Type)
	}
}

type IceOfferCandidate struct {
	Ice string `json:"ice" validate:"required"`
}

func (ice *IceOfferCandidate) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	err := mongoclient.Mongo.UpdateSignalingDoc(*signalingDoc, bson.M{
		"$push": bson.M{
			"offerIce": ice.Ice,
		},
	})
	return nil, err
}

type IceAnswerCandidate struct {
	Ice string `json:"ice" validate:"required"`
}

func (ice *IceAnswerCandidate) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	err := mongoclient.Mongo.UpdateSignalingDoc(*signalingDoc, bson.M{
		"$push": bson.M{
			"answerIce": ice.Ice,
		},
	})
	return nil, err
}

type NewOffer struct {
	Sdp string `json:"sdp" validate:"required"`
}

func (offer *NewOffer) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	err := mongoclient.Mongo.UpdateSignalingDoc(*signalingDoc, bson.M{
		"$set": bson.M{
			"offer": offer.Sdp,
		},
	})
	return nil, err
}

type NewAnswer struct {
	Sdp string `json:"sdp" validate:"required"`
}

func (answer *NewAnswer) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	err := mongoclient.Mongo.UpdateSignalingDoc(*signalingDoc, bson.M{
		"$set": bson.M{
			"answer": answer.Sdp,
		},
	})
	return nil, err
}

type ListenOffersHost struct {
	Url           string `json:"url" validate:"required"`
	PasswordFiles string `json:"passwordFiles" validate:"required"`
}

func (l *ListenOffersHost) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	if !mongoclient.Mongo.IsPasswordFilesValid(l.Url, l.PasswordFiles) {
		return nil, fmt.Errorf("invalid password files")
	}

	mongoclient.Mongo.ListenNewConns(l.Url, func(changes mongoclient.ListenNewConnsEvent) bool {
		for _, msg := range parseUpdatedFields(changes.U) {
			msg.SignalingId = changes.Id.Hex()
			msgBytes, _ := json.Marshal(msg)

			if err := websocket.Message.Send(ws, string(msgBytes)); err != nil {
				fmt.Printf("send msg err4: %v\n", err)
				return false
			}
		}
		return true
	})

	return nil, nil
}

type ListenOffersConn struct{}

func (l ListenOffersConn) Process(ws *websocket.Conn, signalingDoc *string) (interface{}, error) {
	mongoclient.Mongo.ListenSignaling(*signalingDoc, func(changes mongoclient.ListenSignalingEvent) bool {
		for _, msg := range parseUpdatedFields(changes.U) {
			if msg.Type == MsgNewOffer || msg.Type == MsgOfferIceCandidate {
				continue
			}

			msgBytes, _ := json.Marshal(msg)

			if err := websocket.Message.Send(ws, string(msgBytes)); err != nil {
				fmt.Printf("send msg err5: %v\n", err)
				return false
			}
		}
		return true
	})

	return nil, nil
}

type MessageError struct {
	Msg string `json:"msg"`
}

func parseUpdatedFields(u map[string]interface{}) []Message {
	var msgs []Message = []Message{}

	for k, v := range u {
		switch k {
		case "offer":
			data, _ := json.Marshal(NewOffer{Sdp: v.(string)})
			msgs = append(msgs, Message{
				Type: MsgNewOffer,
				Data: data,
			})
		case "answer":
			data, _ := json.Marshal(NewAnswer{Sdp: v.(string)})
			msgs = append(msgs, Message{
				Type: MsgNewAnswer,
				Data: data,
			})
		default:
			if strings.HasPrefix(k, "offerIce.") {
				data, _ := json.Marshal(IceOfferCandidate{Ice: v.(string)})
				msgs = append(msgs, Message{
					Type: MsgOfferIceCandidate,
					Data: data,
				})
			}
			if strings.HasPrefix(k, "answerIce.") {
				data, _ := json.Marshal(IceOfferCandidate{Ice: v.(string)})
				msgs = append(msgs, Message{
					Type: MsgAnswerIceCandidate,
					Data: data,
				})
			}
		}
	}

	return msgs
}
