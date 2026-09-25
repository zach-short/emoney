package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Player struct {
	ID         bson.ObjectID `bson:"_id" json:"id"`
	RoomID     bson.ObjectID `bson:"roomId" json:"roomId"`
	Properties []Property    `bson:"-" json:"properties,omitempty"`
	IsBanker   bool          `bson:"isBanker" json:"isBanker"`
	IsActive   bool          `bson:"isActive" json:"isActive"`
	Balance    int           `bson:"balance" json:"balance"`
	Name       string        `bson:"name" json:"name"`
	Color      string        `bson:"color" json:"color"`
}
