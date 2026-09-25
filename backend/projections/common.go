package projections

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func CreateProjection(fields string, defaultFields []string) bson.M {
	projection := bson.M{}

	for _, field := range defaultFields {
		projection[field] = 1
	}

	if fields != "" {
		for _, field := range strings.Split(fields, ",") {
			projection[strings.TrimSpace(field)] = 1
		}
	}
	return projection
}
