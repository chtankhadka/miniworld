package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/chtan/miniworld/config"
	common_controllers "github.com/chtan/miniworld/controllers/common"
	auth_models "github.com/chtan/miniworld/models/auth"
	"github.com/chtan/miniworld/token"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Authentication is a Gin middleware for JWT validation
func Authentication(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Set a short timeout for database operations
		mctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		clientToken, tokenError := common_controllers.GetMyToken(ctx)
		if tokenError != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": tokenError})
			ctx.Abort()
			return
		}
		// Validate token
		claims, err := token.ValidateToken(clientToken, app)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			ctx.Abort()
			return
		}

		// Optional database verification
		if app.RequireDBCheck {
			filter := bson.M{"_id": claims.UID, "access_token": clientToken}
			var user auth_models.SetSignUpModel
			err := app.Client.Database("miniworld").Collection("users").FindOne(mctx, filter).Decode(&user)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					ctx.JSON(http.StatusUnauthorized, gin.H{"error": "token not found or user unauthorized"})
				} else {
					log.Printf("Database error during token check: %v", err)
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				}
				ctx.Abort()
				return
			}

			// Check if token is revoked
			if user.Revoked {
				ctx.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
				ctx.Abort()
				return
			}
		}

		// Set claims in context for downstream handlers
		ctx.Set("email", claims.Email)
		ctx.Set("_id", claims.UID)

		// Proceed to the next handler
		ctx.Next()
	}
}

// RequireAuthWithRole extends Authentication to enforce role-based access
func RequireAuthWithRole(app *config.AppConfig, requiredRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Run basic authentication first
		Authentication(app)(ctx)
		if ctx.IsAborted() {
			return
		}

		// Example: Check role (assumes roles are stored in DB or token)
		// Here, you'd fetch the user's role from the database or token claims
		uid := ctx.GetString("_id")
		mctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user auth_models.SetSignUpModel
		err := app.Client.Database("miniworld").Collection("users").FindOne(mctx, bson.M{"_id": uid}).Decode(&user)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user role"})
			ctx.Abort()
			return
		}

		// Placeholder: Assume user.Role exists in your User model
		// if user.Role != requiredRole {
		// 	ctx.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		// 	ctx.Abort()
		// 	return
		// }

		ctx.Next()
	}
}
