package device_controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/chtan/miniworld/config"
	device_models "github.com/chtan/miniworld/models/device"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func IAMOnline(app *config.AppConfig, deviceID primitive.ObjectID, isOnline bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := app.Client.Database("miniworld").Collection("devices")

	filter := bson.M{"_id": deviceID}

	update := bson.M{
		"$set": bson.M{
			"is_online":   isOnline,
			"modified_at": time.Now(),
		},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("device not found")
	}

	return nil
}

func RegisterDevice(mctx context.Context, app *config.AppConfig, adminID primitive.ObjectID, details device_models.Device) (device_models.Device, error) {

	coll := app.Client.Database("miniworld").Collection("devices")

	deviceID := primitive.NewObjectID()

	device := device_models.Device{
		ID:         deviceID,
		ClanID:     details.ClanID,
		AdminID:    adminID,
		Password:   details.Password,
		Color:      details.Color,
		IsOnline:   false,
		Updated_At: time.Now(),
	}

	_, err := coll.InsertOne(mctx, device)
	if err != nil {
		return device_models.Device{}, err
	}

	return device, nil
}

func GetDeviceDetails(mctx context.Context, app *config.AppConfig, clientToken string) (*device_models.Device, string) {
	var deviceDetails device_models.Device

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
	err := app.Client.Database("miniworld").Collection("devices").FindOne(mctx, filter, opts).Decode(&deviceDetails)
	if err != nil {
		return nil, err.Error()
	}

	return &deviceDetails, ""
}
