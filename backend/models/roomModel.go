package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Room struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Code    		string             `bson:"code" json:"code"`
	FreeParking int                `bson:"freeParking" json:"freeParking"`
	RoomRules   RoomRules          `bson:"roomRules" json:"roomRules"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
	// Auction is the live auction of a kicked player's estate, or nil when
	// no auction is running in this room. Persisted here rather than in
	// process memory (D18) - see models/auctionModel.go. omitempty on both
	// tags so a room with no auction carries no key at all, which is what
	// handleKickPlayer's conditional write tests for.
	Auction *Auction `bson:"auction,omitempty" json:"auction,omitempty"`
}

type RoomRules struct {
	StartingCash int `bson:"startingCash" json:"startingCash"`
	MaxHouses    int `bson:"maxHouses" json:"maxHouses"`
	MaxHotels    int `bson:"maxHotels" json:"maxHotels"`
}
