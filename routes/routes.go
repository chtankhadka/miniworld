package routes

import (
	"github.com/chtan/miniworld/config"
	"github.com/chtan/miniworld/controllers"
	clan_controllers "github.com/chtan/miniworld/controllers/clan"
	device_controllers "github.com/chtan/miniworld/controllers/device"
	user_controllers "github.com/chtan/miniworld/controllers/user"
	websocket_controllers "github.com/chtan/miniworld/controllers/websocket"
	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes *gin.RouterGroup, app *config.AppConfig) {
	// incomingRoutes.POST("/logout", controllers.Logout(app))
}

func DevicePublicRoutes(incomingRoutes *gin.Engine, app *config.AppConfig) {
	incomingRoutes.POST("/dlogin", device_controllers.LogIn(app))
	incomingRoutes.GET("/api/ws/device", controllers.HandleDeviceWS(app))
	incomingRoutes.GET("/api/ws/devicecam", websocket_controllers.HandleDeviceWSCam(app))

}

func ClanRoutes(incomingRoutes *gin.RouterGroup, app *config.AppConfig) {
	incomingRoutes.POST("/createclan", clan_controllers.CreateClan(app))
	incomingRoutes.POST("/adddevice", clan_controllers.AddDevice(app))
}

func UserPublicRoutes(incomingRoutes *gin.Engine, app *config.AppConfig) {
	// incomingRoutes.POST("/imageverification", controllers.ImageVarification(app))
	incomingRoutes.POST("/usignup", user_controllers.SignUp(app))
	incomingRoutes.POST("/usignin", user_controllers.SignIn(app))
	incomingRoutes.POST("/uvalidateotp", user_controllers.ValidateOtpAndSaveUser(app))

}

func WebSocketRoutes(incomingRoutes *gin.RouterGroup, app *config.AppConfig) {
	incomingRoutes.GET("/ws/user", controllers.HandleUserWS(app))
	incomingRoutes.GET("/ws/usercam", websocket_controllers.HandleUserWSCam(app))

}
