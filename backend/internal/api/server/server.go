package server

import (
	"net/http"

	"github.com/wb-go/wbf/ginext"
)

func New(addr string, router *ginext.Engine) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
