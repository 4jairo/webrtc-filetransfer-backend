package routes

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"

	mongoclient "github.com/4jairo/webrtcFiletransfer/db"
	"github.com/4jairo/webrtcFiletransfer/handler"
	"github.com/4jairo/webrtcFiletransfer/schema"
	"github.com/gorilla/mux"
)

type NewUrlRequest struct {
	Password string        `json:"password"`
	Files    []schema.File `json:"files"`
}

type NewUrlResponse struct {
	Url           string `json:"url"`
	PasswordFiles string `json:"passwordFiles" validate:"required"` // password files
}

func NewFileHandler(req *http.Request, newUrl NewUrlRequest) (*NewUrlResponse, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return nil, errors.New("error while creating url")
	}
	passwordFiles := base64.RawStdEncoding.EncodeToString(bytes)

	filesSchema := schema.NewFileSchema(newUrl.Password, passwordFiles, newUrl.Files, mongoclient.MongoLastUpdateTTL)

	objId, err := mongoclient.Mongo.CreateFilesDoc(filesSchema)
	if err != nil {
		return nil, errors.New("error while creating url")
	}

	return &NewUrlResponse{
		Url:           objId.Hex(),
		PasswordFiles: passwordFiles,
	}, nil
}

//----------------------------------------------------------------------

type AddFileRequest struct {
	Url           string        `json:"url" validate:"required"`
	PasswordFiles string        `json:"passwordFiles" validate:"required"` // password files
	Files         []schema.File `json:"files"`
}

func AddFileHandler(req *http.Request, addFile AddFileRequest) (*any, error) {
	err := mongoclient.Mongo.AddFiles(addFile.Url, addFile.PasswordFiles, addFile.Files)
	return nil, err
}

//----------------------------------------------------------------------

func GetFilesHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	result, err := mongoclient.Mongo.GetFiles(vars["objId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	handler.SendResponse(w, result)
}

// ----------------------------------------------------------------------

type RemoveFilesRequest struct {
	Url           string   `json:"url" validate:"required"`
	PasswordFiles string   `json:"passwordFiles" validate:"required"` // password files
	Files         []string `json:"files" validate:"required"`
}

func RemoveFilesHandler(req *http.Request, removeFile RemoveFilesRequest) (*any, error) {
	err := mongoclient.Mongo.RemoveFiles(removeFile.Url, removeFile.PasswordFiles, removeFile.Files)
	return nil, err
}
