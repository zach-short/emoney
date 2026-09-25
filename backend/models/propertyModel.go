package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Property struct {
	ID               bson.ObjectID `bson:"_id" json:"id"`
	RoomID           bson.ObjectID `bson:"roomId" json:"roomId"`
	PlayerID         bson.ObjectID `bson:"playerId,omitempty" json:"playerId,omitempty"`
	PropertyIndex    int           `bson:"propertyIndex" json:"propertyIndex"`
	HouseCost        int           `bson:"houseCost" json:"houseCost"`
	Group            string        `bson:"group" json:"group"`
	Color            string        `bson:"color" json:"color"`
	Price            int           `bson:"price" json:"price"`
	Name             string        `bson:"name" json:"name"`
	IsMortgaged      bool          `bson:"isMortgaged" json:"isMortgaged"`
	DevelopmentLevel int           `bson:"developmentLevel" json:"developmentLevel"`
	RentPrices       []int         `bson:"rentPrices" json:"rentPrices"`
	Images           []string      `bson:"images" json:"images"`
}

type PropertyDetails []struct {
	PropertyID string
	Count      int
}

type PropertyPurchase struct {
	ID         bson.ObjectID `bson:"_id" json:"id"`
	RoomID     bson.ObjectID `bson:"roomId" json:"roomId"`
	BuyerID    bson.ObjectID `bson:"buyerId" json:"buyerId"`
	PropertyID bson.ObjectID `bson:"propertyId" json:"propertyId"`
	TimeStamp  time.Time     `bson:"timestamp" json:"timestamp"`
	Price      int           `bson:"price" json:"price"`
	Status     string        `bson:"status" json:"status"`
}
