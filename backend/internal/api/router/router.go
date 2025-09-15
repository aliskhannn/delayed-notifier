package router

import (
	"github.com/wb-go/wbf/ginext"

	"github.com/aliskhannn/delayed-notifier/internal/api/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/middlewares"
)

func New(handler *notification.Handler) *ginext.Engine {
	e := ginext.New()
	e.Use(middlewares.CORSMiddleware())
	e.Use(ginext.Logger())
	e.Use(ginext.Recovery())

	api := e.Group("/api/notify")
	{
		api.POST("/", handler.Create)
		api.GET("/", handler.GetAll)
		api.GET("/:id", handler.GetStatus)
		api.DELETE("/:id", handler.Cancel)
	}

	return e
}
