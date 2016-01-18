package phpfpm

import (
	"net/http"

	"github.com/yookoala/gofast"
)

// NewHandler returns a new FastCGI handler
func NewHandler(network, address string) http.Handler {
	return gofast.NewHandler(network, address)
}
