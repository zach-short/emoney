package projections

import "go.mongodb.org/mongo-driver/v2/bson"

func CreatePlayerProjection(fields string) bson.M {
	defaultFields := []string{"_id"}

	return CreateProjection(fields, defaultFields)
}
