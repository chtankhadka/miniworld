package common_controllers

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/chtan/miniworld/config"
	models "github.com/chtan/miniworld/models/common"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	// "go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var uploader *s3manager.Uploader

func init() {
	AWSSession()

}

func Generate_OTP() *int {

	return nil

}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a plain-text password with a stored hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func SendMail(userMail string, message string) bool {
	return true
}

func GetMyToken(ctx *gin.Context) (string, error) {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	fields := strings.Fields(authHeader)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "bearer") {
		return "", fmt.Errorf("invalid authorization format, expected 'Bearer <token>'")
	}

	return fields[1], nil
}

func GetMyId(mctx context.Context, ctx *gin.Context, app *config.AppConfig) (*models.UserDetails, error) {
	var userDetails models.UserDetails
	token, err := GetMyToken(ctx)
	if err != nil {
		return &userDetails, err
	}

	filter := bson.M{
		"access_token": token,
	}

	// // Define the projection to return specific fields
	// opts := options.FindOne().SetProjection(bson.M{
	// 	"user_id":          1,
	// 	"first_name":       1,
	// 	"last_name":        1,
	// 	"email":            1,
	// 	"profile_url":      1,
	// 	"user_interests":   1,
	// 	"user_looking_for": 1,
	// 	"user_history":     1,
	// 	"_id":              0,
	// })

	// Variable to store the result

	// Execute the query
	err = app.Client.Database("miniworld").Collection("users").FindOne(mctx, filter).Decode(&userDetails)

	return &userDetails, err
}

// Success response helper
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(200, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Error:   nil, // Always present but null
	})
}

// Error response helper
func ErrorResponse(c *gin.Context, statusCode int, message string, errorDetails interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Message: message,
		Data:    nil, // Always present but null
		Error:   errorDetails,
	})
}

func UploadUlalaImageAndReturnUrl(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userDetails, idError := GetMyId(mctx, ctx, app)
		if idError != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": idError})
			ctx.Abort()
			return
		}

		file, fileHeader, err := ctx.Request.FormFile("file")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving file"})
			return
		}
		defer file.Close()

		// Example: Do something with the byte array, like saving it
		uploadedURL, err := SaveFileToAWS(file, fileHeader, fmt.Sprintf("%s_%d.jpg", userDetails.ID, time.Now().UnixMilli()))

		// Respond with success and file information
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err})
		} else {
			SuccessResponse(ctx, "Photo Url is here", gin.H{
				"urls": uploadedURL,
			})
		}
	}
}

func SaveFileToAWS(fileReader io.Reader, fileHeader *multipart.FileHeader, pathAndName string) (string, error) {
	// Upload the file to S3 using the fileReader
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("bucketName"),        // Ensure bucketName is set
		Key:    aws.String(fileHeader.Filename), // Use the filename as the S3 object key
		Body:   fileReader,
	})
	if err != nil {
		return "", err
	}

	// Return the URL of the uploaded file
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", "bucketName", pathAndName)
	return url, nil
}

func AWSSession() {

	envError := godotenv.Load()
	if envError != nil {
		log.Fatalf("Error loading .env file: %v", envError)
	}

	AWS_ACCESS_KEY := os.Getenv("AWS_ACCESS_KEY") // Access Key ID from IAM user
	AWS_SECRET_KEY := os.Getenv("AWS_SECRET_KEY") // Secret Access Key from IAM user

	aswSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("eu-north-1"),
			Credentials: credentials.NewStaticCredentials(
				AWS_ACCESS_KEY,
				AWS_SECRET_KEY,
				"",
			),
		},
	})
	if err != nil {
		panic(err)
	}

	uploader = s3manager.NewUploader(aswSession)
}
func ToObjectID(id string) (primitive.ObjectID, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return objID, nil
}
