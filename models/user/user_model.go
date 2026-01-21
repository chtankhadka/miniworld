package user_models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`            // device_id
	ClanID        primitive.ObjectID `json:"clan_id" bson:"clan_id"`   // reference to clan
	AdminID       primitive.ObjectID `json:"admin_id" bson:"admin_id"` // owner/admin user
	Color         string             `json:"color" bson:"color"`
	IsOnline      bool               `json:"is_online" bson:"is_online"`
	Access_Token  string             `json:"access_token" bson:"access_token"`
	Refresh_Token string             `json:"refresh_token" bson:"refresh_token"`
	Refresh_ID    time.Time          `json:"refresh_id" bson:"refresh_id"`
	Created_At    time.Time          `json:"created_at" bson:"created_at"`
	Updated_At    time.Time          `json:"updated_at" bson:"updated_at"`
}
