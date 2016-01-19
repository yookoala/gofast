package gofast

import (
	"bytes"
	"log"
	"net"
	"net/http"
)

// BeforeDo is the function to change FastCGI request before
// the client do it
type BeforeDo func(req *Request, r *http.Request) (*Request, error)

// passthrough is the simplest BeforeDo implementation
func passthrough(req *Request, r *http.Request) (*Request, error) {
	return req, nil
}

// Handler is the interface for a FastCGI
// web server, which proxy request to FastCGI
// application through network port or socket
type Handler interface {
	SetLogger(logger *log.Logger)
	SetBeforeDo(fn BeforeDo)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewHandler returns a new Handler interface
func NewHandler(network, address string) Handler {
	return &defaultHandler{
		network:  network,
		address:  address,
		beforeDo: passthrough,
	}
}

// defaultHandler implements Proxy interface
type defaultHandler struct {
	network  string
	address  string
	beforeDo BeforeDo
	logger   *log.Logger
}

// SetLogger implements Handler
func (h *defaultHandler) SetLogger(logger *log.Logger) {
	h.logger = logger
}

// SetBeforeDo implements Handler
func (h *defaultHandler) SetBeforeDo(beforeDo BeforeDo) {
	if beforeDo != nil {
		h.beforeDo = beforeDo
	}
}

// ServeHTTP implements http.Handler
func (h *defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial(h.network, h.address)
	if err != nil {
		http.Error(w, "failed to connect to FastCGI application", http.StatusBadGateway)
		log.Printf("gofast: unable to connect to FastCGI application "+
			"(network=%#v, address=%#v, error=%#v)",
			h.network, h.address, err.Error())
		return
	}

	c := NewClient(conn, 0)
	req := c.NewRequest(r)

	req, err = h.beforeDo(req, r)
	if err != nil {
		log.Printf("gofast: stopped by beforeDo "+
			"(network=%#v, address=%#v, error=%#v)",
			h.network, h.address, err.Error())
		return
	}

	// handle the result
	resp, err := c.Do(req)
	if err != nil {
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		log.Printf("gofast: unable to process request "+
			"(network=%#v, address=%#v, error=%#v)",
			h.network, h.address, err.Error())
		return
	}
	errBuffer := new(bytes.Buffer)
	resp.WriteTo(w, errBuffer)

	if errBuffer.Len() > 0 {
		log.Printf("gofast: error stream from application process "+
			"(network=%#v, address=%#v, error=%#v)",
			h.network, h.address, errBuffer.String())
		return
	}
}
