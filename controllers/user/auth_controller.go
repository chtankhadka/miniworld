package user_controllers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/chtan/miniworld/config"
	common_controllers "github.com/chtan/miniworld/controllers/common"
	"github.com/chtan/miniworld/helper"
	auth_models "github.com/chtan/miniworld/models/auth"
	common_models "github.com/chtan/miniworld/models/common"
	"github.com/chtan/miniworld/token"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SignUp(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var getSignupDetails auth_models.GetSignUpModel
		if err := ctx.BindJSON(&getSignupDetails); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}

		if helper.IsFieldUsed(app, mctx, ctx, "email", getSignupDetails.Email) {
			return
		}
		password, err := common_controllers.HashPassword(getSignupDetails.Password)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Error In Hashing", err.Error())
			return
		}
		getSignupDetails.Password = password
		getSignupDetails.ID = primitive.NewObjectID()
		getSignupDetails.User_ID = getSignupDetails.ID.Hex()
		getSignupDetails.OTP = rand.Intn(9000) + 1000
		getSignupDetails.Count = 0

		message := fmt.Sprintf("Hello, your OTP is %d. Please keep it confidential.", getSignupDetails.OTP)
		if !common_controllers.SendMail(getSignupDetails.Email, message) {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Error In OTP", "Failed to Send OTP")
			return
		}
		// Create TTL index on tempOtps collection (run once)
		CreateTTLIndex(app.Client.Database("miniworld").Collection("tempData"))
		insertTempErr := InsertTempUsers(app.Client.Database("miniworld").Collection("tempData"), getSignupDetails)
		if insertTempErr != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Temperory users", "Failed to save temporary user")
			return
		}
		common_controllers.SuccessResponse(ctx, "OTP sent and Data stored in Temp", gin.H{"user_id": getSignupDetails.User_ID})
	}
}

func ValidateOtpAndSaveUser(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var validateOTP common_models.ValidateOTP
		if err := ctx.BindJSON(&validateOTP); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}
		objID, err := primitive.ObjectIDFromHex(validateOTP.ID)
		if err != nil {
			fmt.Println("Invalid ID:", err)
			return
		}

		filter := bson.M{
			"_id":   objID,
			"count": bson.M{"$lt": 4},
		}

		var getSignupDetails auth_models.GetSignUpModel
		err = app.Client.Database("miniworld").Collection("tempData").FindOne(mctx, filter).Decode(&getSignupDetails)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				common_controllers.ErrorResponse(ctx, http.StatusNotFound, "Not data found", err.Error())

			} else {
				common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Something else", err.Error())
			}
			return
		}

		if getSignupDetails.OTP != validateOTP.OTP {
			_, updateErr := app.Client.Database("miniworld").Collection("tempData").UpdateOne(
				mctx,
				bson.M{"_id": validateOTP.ID},
				bson.M{"$inc": bson.M{"count": 1}},
			)
			if updateErr != nil {
				common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to update attempt count", updateErr.Error())
				return
			}
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid OTP", "OTP Not matched")
			return
		}

		if helper.IsFieldUsed(app, mctx, ctx, "email", getSignupDetails.Email) {
			return
		}

		var setSignUpModel auth_models.SetSignUpModel
		setSignUpModel.ID = getSignupDetails.ID
		setSignUpModel.Email = getSignupDetails.Email
		setSignUpModel.First_Name = getSignupDetails.First_Name
		setSignUpModel.Last_Name = getSignupDetails.Last_Name
		setSignUpModel.Password = getSignupDetails.Password
		setSignUpModel.IsAccountActive = true
		setSignUpModel.Created_At = time.Now()
		setSignUpModel.Updated_At = time.Now()
		setSignUpModel.Revoked = false

		// Generate initial tokens
		tokenPair, err := token.GenerateTokenPair(setSignUpModel.Email, setSignUpModel.ID, app)
		if err != nil {
			log.Printf("Failed to generate tokens: %v", err)
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to generate tokens", err.Error())
			return
		}

		setSignUpModel.Access_Token = tokenPair.AccessToken
		setSignUpModel.Refresh_Token = tokenPair.RefreshToken

		_, err = app.Client.Database("miniworld").Collection("users").InsertOne(mctx, setSignUpModel)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to save user", err.Error())
			return
		}

		common_controllers.SuccessResponse(ctx, "User Created Successfully", gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"id":            setSignUpModel.ID.Hex(),
		})
	}
}

func SignIn(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var creds struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := ctx.BindJSON(&creds); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}

		var user auth_models.SetSignUpModel
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := app.Client.Database("miniworld").Collection("users").FindOne(mctx, bson.M{"email": creds.Email}).Decode(&user)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials", err.Error())
			return
		}

		if !common_controllers.CheckPasswordHash(creds.Password, user.Password) {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid credentials", "Password Not Matched")
			return
		}

		tokenPair, err := token.GenerateTokenPair(user.Email, user.ID, app)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to generate tokens", err.Error())
			return
		}

		// Update tokens in the database
		_, err = app.Client.Database("miniworld").Collection("users").UpdateOne(
			mctx,
			bson.M{"_id": user.ID},
			bson.M{"$set": bson.M{
				"access_token":  tokenPair.AccessToken,
				"refresh_token": tokenPair.RefreshToken,
				"updated_at":    time.Now(),
				"revoked":       false,
			}},
		)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to update tokens", err.Error())
			return
		}
		common_controllers.SuccessResponse(ctx, "Signed In Successfully", gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"id":            user.ID.Hex(),
		})
	}
}

func MyProfile(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userDetails, idError := common_controllers.GetMyId(mctx, ctx, app)
		if idError != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": idError})
			ctx.Abort()
			return
		}
		common_controllers.SuccessResponse(ctx, "My profile", userDetails)
	}
}

func RefreshToken(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusBadRequest, "Parsing Error", err.Error())
			return
		}

		newTokenPair, err := token.RefreshTokens(req.RefreshToken, app)
		if err != nil {
			common_controllers.ErrorResponse(ctx, http.StatusUnauthorized, "Token Error", err.Error())
			return
		}
		common_controllers.SuccessResponse(ctx, "Token refreshed", gin.H{
			"access_token":  newTokenPair.AccessToken,
			"refresh_token": newTokenPair.RefreshToken,
		})
	}
}

func Logout(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !app.RequireDBCheck {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "logout not supported in stateless mode"})
			return
		}

		uid := ctx.GetString("uid")
		mctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := app.Client.Database("miniworld").Collection("users").UpdateOne(mctx, bson.M{"user_id": uid}, bson.M{
			"$set": bson.M{"revoked": true, "updated_at": time.Now().Unix()},
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
			return
		}
		common_controllers.SuccessResponse(ctx, "logged out successfully", "logged Out")
	}
}

func CreateTTLIndex(collection *mongo.Collection) {
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}}, // Accending index on expireAt
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	_, err := collection.Indexes().CreateOne(context.Background(), index)
	if err != nil {
		log.Fatal("Failed to create TTL index:", err)
	}
	log.Println("TTL index created on expireAt field")
}

func InsertTempUsers(collection *mongo.Collection, userDetails auth_models.GetSignUpModel) error {
	userDetails.Expires_At = time.Now().Add(5 * time.Minute)

	_, err := collection.InsertOne(context.Background(), userDetails)
	return err
}
