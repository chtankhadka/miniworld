package websocket_controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/chtan/miniworld/config"
	common_controllers "github.com/chtan/miniworld/controllers/common"
	device_controllers "github.com/chtan/miniworld/controllers/device"
	user_controllers "github.com/chtan/miniworld/controllers/user"
	"github.com/chtan/miniworld/mywebsocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		// In dev: allow all origins. For prod, restrict this.
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	sessionManager = mywebsocket.NewSessionManager()
)

// ws://server/ws/device
func HandleDeviceWSCam(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1) AUTH BEFORE WEBSOCKET UPGRADE
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clientToken, tokenError := common_controllers.GetMyToken(ctx)
		if tokenError != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": tokenError})
			return
		}
		println("i am here at device cam")

		deviceDetails, idError := device_controllers.GetDeviceDetails(mctx, app, clientToken)
		if idError != "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": idError})
			println(idError)
			return
		}
		println("2")

		deviceID := deviceDetails.ID.Hex()
		if deviceID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "deviceId not found"})
			return
		}

		println("3")
		// 2) UPGRADE TO WEBSOCKET AFTER AUTH
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Println("❌ Device upgrade error:", err)
			device_controllers.IAMOnline(app, deviceDetails.ID, false)
			return
		}
		println("4")

		// 3) REGISTER DEVICE SESSION
		sessionManager.AddDevice(deviceID+"_Cam", conn)
		log.Println("✅ Device Cam connected:", deviceID)
		device_controllers.IAMOnline(app, deviceDetails.ID, true)

		defer func() {
			log.Println("⚠️ cam Device disconnected:", deviceID)
			sessionManager.RemoveDevice(deviceID + "_Cam")
			device_controllers.IAMOnline(app, deviceDetails.ID, false)
			conn.Close()
		}()

		// 4) STREAMING / FORWARDING LOOP
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("⚠️ cam Device read error:", err)
				return
			}
			if msgType != websocket.BinaryMessage {
				continue // ignore text / control frames
			}

			// ONE device -> ONE controlling user, so direct lookup:
			userSession := sessionManager.GetUserByDevice(deviceID + "_Cam")
			if userSession != nil {
				log.Println("data recived from cam and sending to user")
				_ = userSession.Conn.WriteMessage(websocket.BinaryMessage, data)
			}
		}
	}
}

// ===================== USER WEBSOCKET =====================

// ws://server/ws/user?deviceId=<deviceId>
func HandleUserWSCam(app *config.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1) AUTH BEFORE WEBSOCKET UPGRADE
		mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		println("i am 1")

		clientToken, tokenError := common_controllers.GetMyToken(ctx)
		if tokenError != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": tokenError})
			println("i am 2")

			return
		}

		userDetails, idError := user_controllers.GetUserDetails(mctx, app, clientToken)
		println("i am 3")

		if idError != "" {
			println("i am 4")

			ctx.JSON(http.StatusUnauthorized, gin.H{"error": idError})
			return
		}

		userID := userDetails.ID.Hex()
		deviceID := ctx.Query("deviceId") // which device this user wants to control

		if userID == "" || deviceID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing userId or deviceId"})
			return
		}

		// 2) UPGRADE TO WEBSOCKET AFTER AUTH
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Println("❌ User upgrade error:", err)
			return
		}

		// 3) REGISTER USER SESSION (one device -> one user)
		sessionManager.AddUser(userID+"_Cam", deviceID+"_Cam", conn)
		log.Printf("✅ User %s connected, controlling device cam %s\n", userID, deviceID)

		defer func() {
			log.Println("⚠️ User disconnected:", userID)
			sessionManager.RemoveUser(userID + "_Cam")
			conn.Close()
		}()

		// 4) STREAMING / FORWARDING LOOP
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				log.Println("⚠️ User read error:", err)
				return
			}
			if msgType != websocket.BinaryMessage {
				continue // ignore text / control frames
			}

			// Forward to THIS user's device only
			devConn := sessionManager.GetDeviceConn(deviceID + "_Cam")
			if devConn != nil {
				_ = devConn.WriteMessage(websocket.BinaryMessage, data)
			}
		}
	}
}
