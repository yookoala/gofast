package python3

import (
	"net/http"

	"github.com/yookoala/gofast"
)

// NewHandler returns a new FastCGI handler
func NewHandler(entrypoint, network, address string) http.Handler {
	h := gofast.NewHandler(gofast.NewFileEndpoint(entrypoint), network, address)
	return h
}
