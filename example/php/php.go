package php

import (
	"net/http"

	"github.com/yookoala/gofast"
)

// NewHandler returns a new FastCGI handler
func NewHandler(root, network, address string) http.Handler {
	h := gofast.NewHandler(gofast.NewPHPFS(root), network, address)
	return h
}
