package user_controllers

import (
	"context"

	"github.com/chtan/miniworld/config"
	user_models "github.com/chtan/miniworld/models/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetUserDetails(mctx context.Context, app *config.AppConfig, clientToken string) (*user_models.User, string) {
	var userDetails user_models.User

	filter := bson.M{
		"access_token": clientToken,
	}

	// Define the projection to return specific fields
	opts := options.FindOne().SetProjection(bson.M{
		"_id":     1,
		"clan_id": 1,
	})

	// Variable to store the result

	// Execute the query
	err := app.Client.Database("miniworld").Collection("users").FindOne(mctx, filter, opts).Decode(&userDetails)
	if err != nil {
		return nil, err.Error()
	}

	return &userDetails, ""
}

func GetOnlineDevices() {

}
