package proxy

import (
	"bytes"
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

	c := gofast.NewClient(conn, 0)
	req := c.NewRequest(r)

	// some input for req
	req.Params["hello"] = "world"
	req.Params["foo"] = "bar"

	// FIXME: Stdin should be a Reader / ReadCloser
	//        instead of []byte.
	//
	//        Pass the r.Body (ReadCloser) to
	//        modified version of writeRecord
	//        instead of storing another copy of body
	//        in memory
	req.Stdin = []byte(r.Form.Encode())

	// handle the result
	resp, err := c.Do(req)
	if err != nil {
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		log.Printf("gofast: unable to process request "+
			"(network=%#v, address=%#v, error=%#v)",
			p.network, p.address, err.Error())
		return
	}
	errBuffer := new(bytes.Buffer)
	resp.WriteTo(w, errBuffer)

	if errBuffer.Len() > 0 {
		log.Printf("gofast: error stream from application process "+
			"(network=%#v, address=%#v, error=%#v)",
			p.network, p.address, errBuffer.String())
		return
	}
}
