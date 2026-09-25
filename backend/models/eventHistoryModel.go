package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type EventHistory struct {
	ID        bson.ObjectID `bson:"_id" json:"id"`
	RoomID    bson.ObjectID `bson:"roomId" json:"roomId"`
	TimeStamp time.Time     `bson:"timestamp" json:"timestamp"`
	Event     string        `bson:"event" json:"event"` // description of what happened
	EventType []string      `bson:"eventType" json:"eventType"`
}
