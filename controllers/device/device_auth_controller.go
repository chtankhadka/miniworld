package device_controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/chtan/miniworld/config"
	common_controllers "github.com/chtan/miniworld/controllers/common"
	device_models "github.com/chtan/miniworld/models/device"
	"github.com/chtan/miniworld/token"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LogIn(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var device_request struct {
			ID       primitive.ObjectID `json:"id" bson:"_id"`
			Password string             `json:"password" bson:"password"`
		}
		if err := ctx.BindJSON(&device_request); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}

		var device device_models.Device
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := app.Client.Database("miniworld").Collection("devices").FindOne(mctx, bson.M{"_id": device_request.ID}).Decode(&device)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials", err.Error())
			return
		}

		if !common_controllers.CheckPasswordHash(device_request.Password, device.Password) {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials", "Password Not Matched")
			return
		}

		tokenPair, err := token.GenerateTokenPair(device.ID.Hex(), device.AdminID, app)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to generate tokens", err.Error())
			return
		}

		// Update tokens in the database
		_, err = app.Client.Database("miniworld").Collection("devices").UpdateOne(
			mctx,
			bson.M{"_id": device.ID},
			bson.M{"$set": bson.M{
				"access_token":  tokenPair.AccessToken,
				"refresh_token": tokenPair.RefreshToken,
				"updated_at":    time.Now(),
				"is_online":     true,
				"revoked":       false,
			}},
		)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to update tokens", err.Error())
			return
		}

		err = IAMOnline(app, device_request.ID, true)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials", err.Error())
			return

		}
		common_controllers.SuccessResponse(ctx, "Signed In Successfully", gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"id":            device.ID.Hex(),
		})
	}
}
