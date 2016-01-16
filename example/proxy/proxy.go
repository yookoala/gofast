package proxy

import (
	"net/http"

	"github.com/yookoala/gofast"
)

// New returns a new Proxy interface
func New(pass string) Proxy {
	c := gofast.NewClient(pass)
	return &proxy{c}
}

type proxy struct {
	c gofast.Client
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fcgir := gofast.NewRequest()
	_ = fcgir
}

// Proxy is the interface for a FastCGI proxy
type Proxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
