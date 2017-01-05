package gofast

import (
	"bytes"
	"log"
	"net"
	"net/http"
)

// Handler is the interface for a FastCGI
// web server, which proxy request to FastCGI
// application through network port or socket
type Handler interface {
	SetLogger(logger *log.Logger)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewHandler returns a new Handler interface
func NewHandler(middleware Middleware, network, address string) Handler {
	sessionHandler := BasicSession
	if middleware != nil {
		sessionHandler = middleware(sessionHandler)
	}
	return &defaultHandler{
		sessionHandler: sessionHandler,
		network:        network,
		address:        address,
	}
}

// defaultHandler implements Proxy interface
type defaultHandler struct {
	sessionHandler SessionHandler
	network        string
	address        string
	logger         *log.Logger
}

// SetLogger implements Handler
func (h *defaultHandler) SetLogger(logger *log.Logger) {
	h.logger = logger
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

	// TODO: reuse client for multiple connections (?)
	c := NewClient(conn, 0)

	// handle the session
	resp, err := h.sessionHandler.Handle(c, c.NewRequest(r))
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
