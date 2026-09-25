package manager

import (
	"context"
	"fmt"

	"github.com/zachmshort/emoney-backend/config"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func UpdatePlayerBalance(playerID bson.ObjectID, amount int) error {
	playersColl := config.DB.Collection("Player")
	filter := bson.M{"_id": playerID}
	update := bson.M{"$inc": bson.M{"balance": -amount}}

	_, err := playersColl.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return fmt.Errorf("failed to update player balance: %w", err)
	}

	return nil
}
