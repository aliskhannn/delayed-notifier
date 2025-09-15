package server

import (
	"net/http"

	"github.com/wb-go/wbf/ginext"
)

// New creates a new HTTP server with the specified address and router.
//
// Parameters:
//   - addr: the address (host:port) where the server will listen.
//   - router: the Gin engine (or ginext.Engine) that will handle incoming requests.
//
// Returns:
//   - an *http.Server configured with the given address and router.
func New(addr string, router *ginext.Engine) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
