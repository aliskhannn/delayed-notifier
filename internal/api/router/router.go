package router

import (
	"github.com/wb-go/wbf/ginext"

	"github.com/aliskhannn/delayed-notifier/internal/api/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/middlewares"
)

// New creates a new Gin engine with routes and middlewares for the notification API.
//
// It applies standard middlewares (CORS, logging, recovery) and sets up the
// /api/notify group with the following routes:
//   - POST   /api/notify/       -> handler.Create
//   - GET    /api/notify/       -> handler.GetAll
//   - GET    /api/notify/:id    -> handler.GetStatus
//   - DELETE /api/notify/:id    -> handler.Cancel
func New(handler *notification.Handler) *ginext.Engine {
	// Create a new Gin engine using the extended gin wrapper.
	e := ginext.New()

	// Apply middlewares: CORS, logger, and recovery.
	e.Use(middlewares.CORSMiddleware())
	e.Use(ginext.Logger())
	e.Use(ginext.Recovery())

	// Create an API group for notifications.
	api := e.Group("/api/notify")
	{
		api.POST("/", handler.Create)
		api.GET("/", handler.GetAll)
		api.GET("/:id", handler.GetStatus)
		api.DELETE("/:id", handler.Cancel)
	}

	return e
}
