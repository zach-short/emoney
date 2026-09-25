package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var RoomCollection *mongo.Collection

func CreateRoomInDB(name string, code string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	room := bson.M{
		"name":      name,
		"code":      code,
		"createdAt": time.Now().Unix(),
	}

	result, err := RoomCollection.InsertOne(ctx, room)
	if err != nil {
		return nil, err
	}

	return result.InsertedID, nil
}
