package clan_models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Clan struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`            // Clan ID
	AdminID     primitive.ObjectID `json:"admin_id" bson:"admin_id"` // User ID of clan admin
	ClanDetails *ClanDetails       `json:"clan_details" bson:"clan_details"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type ClanDetails struct {
	Name        string    `json:"name" bson:"name" binding:"required,min=2,max=24"`
	Tag         string    `json:"tag" bson:"tag" binding:"required,len=3,alphanum,uppercase"`
	Description *string   `json:"description" bson:"description"`
	DeviceIDs   *[]string `json:"device_ids" bson:"device_ids"`
}
