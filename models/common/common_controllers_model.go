package common_models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserDetails struct {
	ID             primitive.ObjectID `json:"_id" bson:"_id"`
	FirstName      string             `json:"first_name" bson:"first_name"`
	LastName       string             `json:"last_name" bson:"last_name"`
	Email          string             `json:"email" bson:"email"`
	Profile        string             `json:"profile" bson:"profile"`
	UserInterests  *[]string          `json:"user_interests" bson:"user_interests"`
	UserLookingFor *[]string          `json:"user_looking_for" bson:"user_looking_for"`
	UserHistories  *[]string          `json:"user_history" bson:"user_history"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   interface{} `json:"error"`
}
