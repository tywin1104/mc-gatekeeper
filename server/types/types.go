package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WhitelistRequest represent a whitelist request issued by the requester player
type WhitelistRequest struct {
	ID                 primitive.ObjectID `bson:"_id" json:"_id" json:",omitempty"`
	Username           string             `bson:"username" json:"username" json:",omitempty"`
	Email              string             `bson:"email" json:"email" json:",omitempty"`
	Age                int64              `bson:"age" json:"age" json:",omitempty"`
	Gender             string             `bson:"gender" json:"gender" json:",omitempty"`
	ApplicationText    string             `bson:"applicationText" json:"applicationText"`
	Status             string             `bson:"status" json:"status" json:",omitempty"`
	Timestamp          time.Time          `bson:"timestamp", json:"timestamp" json:",omitempty"`
	processedTimestamp time.Time          `bson:"processedTimestamp", json:"processedTimestamp" json:",omitempty"`
	admin              string             `bson:"admin", json:"admin" json:",omitempty"`
}
