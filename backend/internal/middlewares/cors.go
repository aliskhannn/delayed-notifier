package middlewares

import (
	"net/http"

	"github.com/wb-go/wbf/ginext"
)

// CORSMiddleware returns a Gin middleware that handles Cross-Origin Resource Sharing (CORS) settings.
//
// It sets the necessary headers to allow requests from the frontend (http://localhost:3000)
// and supports credentials, custom headers, and standard HTTP methods.
func CORSMiddleware() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
