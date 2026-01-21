package clan_controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/chtan/miniworld/config"
	common_controllers "github.com/chtan/miniworld/controllers/common"
	device_controllers "github.com/chtan/miniworld/controllers/device"
	clan_models "github.com/chtan/miniworld/models/clan"
	device_models "github.com/chtan/miniworld/models/device"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateClan(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var clan_details clan_models.ClanDetails

		if err := ctx.ShouldBindJSON(&clan_details); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}

		// Get authenticated user
		userDetails, err := common_controllers.GetMyId(mctx, ctx, app)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Build the clan document
		clan := clan_models.Clan{
			ID:          primitive.NewObjectID(),
			AdminID:     userDetails.ID,
			ClanDetails: &clan_details,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Insert into MongoDB
		result, err := app.Client.Database("miniworld").
			Collection("clans").
			InsertOne(mctx, clan)

		if err != nil {
			// Most important: handle duplicate clan name/tag properly!
			if mongo.IsDuplicateKeyError(err) {
				common_controllers.ErrorResponse(ctx, http.StatusConflict, "Clan already exists (name or tag taken)", err.Error())
				return
			}

			// Other MongoDB errors
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create clan", err.Error())
			return
		}

		common_controllers.SuccessResponse(ctx, "Clan created successfully", result)
	}
}

func AddDevice(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var device device_models.Device
		if err := ctx.BindJSON(&device); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}
		password, err := common_controllers.HashPassword(device.Password)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Error In Hashing", err.Error())
			return
		}
		device.Password = password

		userDetails, err := common_controllers.GetMyId(mctx, ctx, app)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		device, err = device_controllers.RegisterDevice(mctx, app, userDetails.ID, device)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		common_controllers.SuccessResponse(ctx, "Token refreshed", gin.H{
			"access_token": "",
		})
	}

}
