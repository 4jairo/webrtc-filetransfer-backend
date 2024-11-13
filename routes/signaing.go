package routes

import (
	"fmt"
	"net/http"

	mongoclient "github.com/4jairo/webrtc-filetransfer-backendBackend/db"
	"github.com/4jairo/webrtc-filetransfer-backendBackend/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NewSignalingRequest struct {
	Url          string `json:"url"`
	PasswordUser string `json:"passwordUser"`
}

type NewSignalingResponse struct {
	Id string `json:"id"`
}

func NewSignalingHandler(req *http.Request, params NewSignalingRequest) (*NewSignalingResponse, error) {
	if !mongoclient.Mongo.IsPasswordUserValid(params.Url, params.PasswordUser) {
		return nil, fmt.Errorf("invalid password")
	}

	objId, err := primitive.ObjectIDFromHex(params.Url)
	if err != nil {
		return nil, err
	}

	signalingDoc := schema.NewSignalingSchema(objId)

	id, err := mongoclient.Mongo.CreateSignalingDoc(signalingDoc)
	if err != nil {
		return nil, err
	}

	return &NewSignalingResponse{
		Id: id.Hex(),
	}, nil
}
