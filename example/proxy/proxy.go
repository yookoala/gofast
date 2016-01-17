package proxy

import (
	"log"
	"net"
	"net/http"

	"github.com/yookoala/gofast"
)

// Proxy is the interface for a FastCGI proxy
type Proxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// New returns a new Proxy interface
func New(network, address string) Proxy {
	return &proxy{
		network: network,
		address: address,
	}
}

// proxy implements Proxy interface
type proxy struct {
	network string
	address string
}

// ServeHTTP implements http.Handler
func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial(p.network, p.address)
	if err != nil {
		http.Error(w, "failed to connect to FastCGI application", http.StatusBadGateway)
		log.Printf("gofast: unable to connect to FastCGI application "+
			"(network=%#v, address=%#v, error=%#v)",
			p.network, p.address, err.Error())
		return
	}

	c := gofast.NewClient(conn)
	req := c.NewRequest()

	// some input for req

	err = c.Handle(req)
	if err != nil {
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		log.Printf("gofast: unable to process request "+
			"(network=%#v, address=%#v, error=%#v)",
			p.network, p.address, err.Error())
		return
	}
}
