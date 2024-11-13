package schema

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FilesCollection string = "files"

type File struct {
	Name         string `json:"name"`
	Length       uint64 `json:"length"`
	LastModified uint64 `json:"lastModified"`
}

type FilesSchema struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	PasswordUser  string             `bson:"passwordUser"`
	PasswordFiles string             `bson:"passwordFiles" validate:"required"`
	Files         []File             `bson:"files"`
	ExpireAt      time.Time          `bson:"expireAt"`
}

func NewFileSchema(passwordUser string, passwordFiles string, files []File, ttl time.Duration) FilesSchema {
	return FilesSchema{
		PasswordUser:  passwordUser,
		PasswordFiles: passwordFiles,
		Files:         files,
		ExpireAt:      time.Now().Add(ttl),
	}
}

const SignalingCollection string = "signaling"

type Candidate struct {
	Sdp string `bson:"sdp, omitempty"`
}

type SignalingSchema struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	FilesId   primitive.ObjectID `bson:"filesId,omitempty"`
	Offer     string             `bson:"offer,omitempty"`
	OfferIce  []string           `bson:"offerIce,omitempty"`
	Answer    string             `bson:"answer,omitempty"`
	AnswerIce []string           `bson:"answerIce,omitempty"`
}

func NewSignalingSchema(filesId primitive.ObjectID) SignalingSchema {
	return SignalingSchema{
		FilesId: filesId,
		// OfferIce:  []string{},
		// AnswerIce: []string{},
	}
}
