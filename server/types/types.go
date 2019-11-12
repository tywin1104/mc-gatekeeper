package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WhitelistRequest represent a whitelist request issued by the requester player
type WhitelistRequest struct {
	ID                 primitive.ObjectID     `bson:"_id" json:"_id"`
	Username           string                 `bson:"username" json:"username"`
	Email              string                 `bson:"email" json:"email"`
	Age                int64                  `bson:"age" json:"age"`
	Gender             string                 `bson:"gender" json:"gender"`
	Status             string                 `bson:"status" json:"status"`
	Timestamp          time.Time              `bson:"timestamp" json:"timestamp"`
	ProcessedTimestamp time.Time              `bson:"processedTimestamp" json:"processedTimestamp" json:",omitempty"`
	Admin              string                 `bson:"admin" json:"admin" json:",omitempty"`
	Note               string                 `bson:"note" json:"note" json:",omitempty"`
	Info               map[string]interface{} `bson:"info" json:"info" json:",omitempty"`
	Assignees          []string               `bson:"assignees" json:"assignees" json:",omitempty"`
}
