package router

import (
	"github.com/wb-go/wbf/ginext"

	"github.com/aliskhannn/delayed-notifier/internal/api/handlers/notification"
)

func New(handler *notification.Handler) *ginext.Engine {
	e := ginext.New()
	e.Use(ginext.Logger())
	e.Use(ginext.Recovery())

	api := e.Group("/api/notify")

	api.POST("/", handler.Create)
	api.GET("/:id", handler.GetStatus)
	api.DELETE("/:id", handler.Cancel)

	return e
}
