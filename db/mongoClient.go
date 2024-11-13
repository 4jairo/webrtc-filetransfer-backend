package mongoclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/4jairo/webrtc-filetransfer-backendBackend/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	client *mongo.Database
}

// const MongoURI string = "mongodb://localhost:27017"
const MongoURI string = "mongodb://mongo:27017"
const MongoDbName string = "webrtc-filetransfer"
const MongoReplicaSet string = "rs0"
const MongoLastUpdateTTL time.Duration = time.Hour * 24

var Mongo MongoClient

func Connect() {
	options := options.Client().
		ApplyURI(MongoURI).
		SetReplicaSet(MongoReplicaSet)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	client, err := mongo.Connect(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	if client.Ping(ctx, nil) != nil {
		log.Fatal("Ping to database failed")
	}

	fmt.Println("Connected to MongoDB (" + MongoURI + "/" + MongoDbName + ")")
	Mongo = MongoClient{
		client: client.Database(MongoDbName),
	}
}

func createDoc(col *mongo.Collection, doc any) (*primitive.ObjectID, error) {
	result, err := col.InsertOne(context.TODO(), doc)
	if err != nil {
		return nil, err
	}

	objId := result.InsertedID.(primitive.ObjectID)
	return &objId, nil
}

func deleteDoc(col *mongo.Collection, id string) error {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id": objId,
	}

	_, err = col.DeleteOne(context.TODO(), filter)
	return err
}

func isPasswordValid(col *mongo.Collection, id string, password string, fieldName string) bool {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false
	}

	filter := bson.M{
		"_id":     objId,
		fieldName: password,
	}

	count, err := col.CountDocuments(context.TODO(), filter)
	if err != nil {
		return false
	}

	return count != 0
}

func (c *MongoClient) CreateFilesDoc(doc schema.FilesSchema) (*primitive.ObjectID, error) {
	col := c.client.Collection(schema.FilesCollection)
	return createDoc(col, doc)
}

func (c *MongoClient) CreateSignalingDoc(doc schema.SignalingSchema) (*primitive.ObjectID, error) {
	col := c.client.Collection(schema.SignalingCollection)
	return createDoc(col, doc)
}

func (c *MongoClient) DeleteFilesDoc(id string) error {
	col := c.client.Collection(schema.FilesCollection)
	return deleteDoc(col, id)
}

func (c *MongoClient) DeleteSignalingDoc(id string) error {
	col := c.client.Collection(schema.SignalingCollection)
	return deleteDoc(col, id)
}

func (c *MongoClient) IsPasswordFilesValid(id string, passwordFiles string) bool {
	col := c.client.Collection(schema.FilesCollection)
	return isPasswordValid(col, id, passwordFiles, "passwordFiles")
}

func (c *MongoClient) IsPasswordUserValid(url string, passwordUser string) bool {
	col := c.client.Collection(schema.FilesCollection)
	return isPasswordValid(col, url, passwordUser, "passwordUser")
}

func (c *MongoClient) AddFiles(id string, passwordFiles string, file []schema.File) error {
	col := c.client.Collection(schema.FilesCollection)

	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":           objId,
		"passwordFiles": passwordFiles,
	}
	update := bson.M{
		"$push": bson.M{
			"files": bson.M{
				"$each": file,
			},
		},
	}

	return col.FindOneAndUpdate(context.TODO(), filter, update).Err()
}

func (c *MongoClient) GetFiles(id string) (*[]schema.File, error) {
	col := c.client.Collection(schema.FilesCollection)

	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"_id": objId,
	}
	findOptions := options.FindOne().SetProjection(bson.M{
		"files": 1,
		"_id":   0,
	})

	var result struct {
		Files []schema.File `bson:"files"`
	}

	errResult := col.FindOne(context.TODO(), filter, findOptions).Decode(&result)
	if errResult != nil {
		return nil, errResult
	}

	return &result.Files, nil
}

func (c *MongoClient) RemoveFiles(id string, passwordFiles string, files []string) error {
	col := c.client.Collection(schema.FilesCollection)

	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":           objId,
		"passwordFiles": passwordFiles,
	}

	update := bson.M{
		"$pull": bson.M{
			"files": bson.M{
				"name": bson.M{
					"$in": files,
				},
			},
		},
	}

	return col.FindOneAndUpdate(context.TODO(), filter, update).Err()
}

func (c *MongoClient) UpdateSignalingDoc(id string, update bson.M) error {
	col := c.client.Collection(schema.SignalingCollection)

	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id": objId,
	}

	return col.FindOneAndUpdate(context.TODO(), filter, update).Err()
}

func listenFor[T any](c *MongoClient, pipeline mongo.Pipeline, cb func(changes T) bool) error {
	col := c.client.Collection(schema.SignalingCollection)

	watchOptions := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	changeStream, err := col.Watch(context.TODO(), pipeline, watchOptions)
	if err != nil {
		return err
	}

	go func() {
		for changeStream.Next(context.TODO()) {
			var changeEvent T
			if err := changeStream.Decode(&changeEvent); err != nil {
				return
			}
			if !cb(changeEvent) {
				return
			}
		}
	}()

	return nil
}

type ListenSignalingEvent struct {
	U map[string]interface{} `bson:"u" json:"u"`
}

// listend to signaling doc with the _id equal to the signalingId
func (c *MongoClient) ListenSignaling(signalingId string, cb func(changes ListenSignalingEvent) bool) error {
	objId, err := primitive.ObjectIDFromHex(signalingId)
	if err != nil {
		return err
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{
				Key: "$match", Value: bson.M{
					"documentKey._id": objId,
					"operationType":   "update",
				},
			},
		},
		bson.D{
			{
				Key: "$addFields", Value: bson.M{
					"u": "$updateDescription.updatedFields",
				},
			},
		},
		bson.D{
			{
				Key: "$project", Value: bson.M{
					"u": bson.M{
						"$arrayToObject": bson.M{
							"$map": bson.M{
								"input": bson.M{"$objectToArray": "$u"},
								"as":    "updated",
								"in": bson.M{
									"k": bson.M{
										"$cond": bson.A{
											bson.M{"$eq": bson.A{bson.M{"$type": "$$updated.v"}, "array"}},
											bson.M{"$concat": bson.A{"$$updated.k", ".0"}},
											"$$updated.k",
										},
									},
									"v": bson.M{
										"$cond": bson.A{
											bson.M{"$eq": bson.A{bson.M{"$type": "$$updated.v"}, "array"}},
											bson.M{"$arrayElemAt": bson.A{"$$updated.v", 0}},
											"$$updated.v",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return listenFor(c, pipeline, cb)
}

type ListenNewConnsEvent struct {
	Id primitive.ObjectID     `bson:"id" json:"id"`
	U  map[string]interface{} `bson:"u" json:"u"`
}

// listens for new signaling docs with the filesId equal to the objId
func (c *MongoClient) ListenNewConns(url string, cb func(changes ListenNewConnsEvent) bool) error {
	objId, err := primitive.ObjectIDFromHex(url)
	if err != nil {
		return err
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{
				Key: "$match", Value: bson.M{
					"fullDocument.filesId": objId,
					"operationType":        "update",
				},
			},
		},
		bson.D{
			{
				Key: "$addFields", Value: bson.M{
					"u":  "$updateDescription.updatedFields",
					"id": "$fullDocument._id",
				},
			},
		},
		bson.D{
			{
				Key: "$project", Value: bson.M{
					"id": 1,
					"u": bson.M{
						"$arrayToObject": bson.M{
							"$map": bson.M{
								"input": bson.M{"$objectToArray": "$u"},
								"as":    "updated",
								"in": bson.M{
									"k": bson.M{
										"$cond": bson.A{
											bson.M{"$eq": bson.A{bson.M{"$type": "$$updated.v"}, "array"}},
											bson.M{"$concat": bson.A{"$$updated.k", ".0"}},
											"$$updated.k",
										},
									},
									"v": bson.M{
										"$cond": bson.A{
											bson.M{"$eq": bson.A{bson.M{"$type": "$$updated.v"}, "array"}},
											bson.M{"$arrayElemAt": bson.A{"$$updated.v", 0}},
											"$$updated.v",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return listenFor(c, pipeline, cb)
}

func (c *MongoClient) ListenFor(pipeline mongo.Pipeline, cb func(changes bson.M) bool) {
	if err := listenFor(c, mongo.Pipeline{
		bson.D{
			{
				Key: "$match", Value: bson.M{
					"operationType": "delete",
				},
			},
		},
	}, cb); err != nil {
		fmt.Printf("err3: %v\n", err)
	}
}

// func (c *MongoClient) CreateTTLIndex(collName string) error {
// 	collection := c.client.Collection(collName)
// 	ttlIndexName := "files_1day_ttl"

// 	// if existent, do not create
// 	indexes, errIndexes := collection.Indexes().List(context.TODO(), nil)
// 	if errIndexes != nil {
// 		return errIndexes
// 	}
// 	for indexes.Next(context.TODO()) {
// 		var idx bson.M
// 		indexes.Decode(&idx)

// 		if idx["name"] != nil && idx["name"] == ttlIndexName {
// 			return nil
// 		}
// 	}

// 	// doesn't exist, Create.
// 	indexModel := mongo.IndexModel{
// 		Keys:    bson.M{"expireAt": 1},
// 		Options: options.Index().SetExpireAfterSeconds(0).SetName("files_1day_ttl"),
// 	}

// 	_, err := collection.Indexes().CreateOne(context.TODO(), indexModel)
// 	if err != nil {
// 		return fmt.Errorf("could not create TTL index: %v", err)
// 	}
// 	return nil
// }

// func (c *MongoClient) UpdateTTL(collName string, documentID string, duration time.Duration) error {
// 	collection := c.client.Collection(collName)

// 	objectID, err := primitive.ObjectIDFromHex(documentID)
// 	if err != nil {
// 		return err
// 	}

// 	filter := bson.M{"_id": objectID}
// 	update := bson.M{
// 		"$set": bson.M{
// 			"expireAt": time.Now().Add(duration),
// 		},
// 	}

// 	_, err = collection.UpdateOne(context.TODO(), filter, update)
// 	return err
// }
