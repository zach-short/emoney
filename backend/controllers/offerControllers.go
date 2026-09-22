package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetPendingOffers is the inbox read: every PENDING offer in the room that
// the named player made or was made to, newest first. It is a REST route
// rather than a websocket reply because the inbox has to be fetchable on
// mount - an offer lives in Mongo precisely so that a reload mid-offer, or a
// dropped OFFER_RECEIVED frame, does not lose it - and every other on-mount
// read in this app is REST through axios (frontend/lib/utils/api.service.ts).
//
// It is its own route rather than a field folded into GetPlayersInRoom,
// deliberately: that response goes to every client in the room, and an offer
// between two players is not room news (websocket/offers.go). Anyone holding
// the room code can still call this for any playerId - a room code is the
// only credential here, settled - but the room-wide read does not carry every
// pending negotiation to every screen by default.
//
// Only PENDING is returned. A settled trade is in the event history; a
// declined or countered one is nothing a screen needs to draw.
func GetPendingOffers(c *gin.Context) {
	code := c.Param("code")
	playerIdStr := c.Query("playerId")

	playerObjID, err := primitive.ObjectIDFromHex(playerIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player ID"})
		return
	}

	var room models.Room
	if err := config.DB.Collection("Room").FindOne(c, bson.M{"code": code}).Decode(&room); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	// make, not var: cursor.All keeps a non-nil empty slice non-nil, so a
	// player with no offers gets [] rather than null and the browser's
	// .filter and .length need no guard.
	offers := make([]models.Offer, 0)
	cursor, err := config.DB.Collection("Offer").Find(c,
		bson.M{
			"roomId": room.ID,
			"status": models.OfferPending,
			"$or": []bson.M{
				{"fromPlayerId": playerObjID},
				{"toPlayerId": playerObjID},
			},
		},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get offers"})
		return
	}
	if err := cursor.All(c, &offers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode offers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

// EnsureOfferIndexes creates the two indexes the inbox read above and the
// counter pin in websocket/offers.go filter on: an offer is always looked up
// by room and by one of its two players. Called once at startup from main.go;
// CreateMany is idempotent for an index that already exists with the same
// keys, so a redeploy costs nothing. This is the first index the app creates
// anywhere (grepped `Indexes()` across backend/ 2026-09-17: nothing), which is
// why it has its own function and its own comment rather than a line inside
// ConnectDB - config/ knows nothing about collections.
func EnsureOfferIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := config.DB.Collection("Offer").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "roomId", Value: 1}, {Key: "toPlayerId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "roomId", Value: 1}, {Key: "fromPlayerId", Value: 1}, {Key: "status", Value: 1}}},
	})
	return err
}
