package gofast_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yookoala/gofast"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newApp(network, address string, fn http.HandlerFunc) (l net.Listener, err error) {
	l, err = net.Listen(network, address)
	if err != nil {
		return
	}
	go fcgi.Serve(l, http.HandlerFunc(fn))
	return
}

func TestClient_NewRequest(t *testing.T) {

	t.Logf("default limit: %d", 65535)

	c, _ := gofast.SimpleClientFactory(
		func() (net.Conn, error) { return nil, nil }, // dummy conn factory
		0,
	)()

	for i := uint32(0); i <= 65535; i++ {
		r := gofast.NewRequest(c, nil)
		if want, have := uint16(i), r.ID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if client can allocate new request
	// when all request ids are already allocated
	newAlloc := make(chan uint16)
	go func(c gofast.Client, newAlloc chan<- uint16) {
		r := gofast.NewRequest(c, nil) // should be blocked before releaseID call
		newAlloc <- r.ID
	}(c, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(65535))
	go func(c gofast.Client, released uint16) {
		c.ReleaseID(released)
	}(c, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}

func TestClient_NewRequestWithLimit(t *testing.T) {

	limit := uint32(rand.Int31n(100) + 10)
	t.Logf("random limit: %d", limit)

	c, _ := gofast.SimpleClientFactory(
		func() (net.Conn, error) { return nil, nil }, // dummy conn factory
		limit,
	)()

	for i := uint32(0); i < limit; i++ {
		r := gofast.NewRequest(c, nil)
		if want, have := uint16(i), r.ID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if client can allocate new request
	// when all request ids are already allocated
	newAlloc := make(chan uint16)
	go func(c gofast.Client, newAlloc chan<- uint16) {
		r := gofast.NewRequest(c, nil) // should be blocked before releaseID call
		newAlloc <- r.ID
	}(c, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(int32(limit)))
	go func(c gofast.Client, released uint16) {
		c.ReleaseID(released)
	}(c, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}

func TestClient_canceled(t *testing.T) {

	// proxy implements Proxy interface
	type proxy struct {
		network string
		address string
	}

	NewRequest := func(c gofast.Client, r *http.Request) (req *gofast.Request) {
		var isHTTPS string
		if r.URL.Scheme == "https" || r.URL.Scheme == "wss" {
			isHTTPS = "on"
		}

		remoteAddr, remotePort, _ := net.SplitHostPort(r.RemoteAddr)
		_, serverPort, err := net.SplitHostPort(r.URL.Host)
		if err != nil {
			if r.URL.Scheme == "https" || r.URL.Scheme == "wss" {
				serverPort = "443"
			} else {
				serverPort = "80"
			}
		}

		req = gofast.NewRequest(c, r)
		req.Params["CONTENT_TYPE"] = r.Header.Get("Content-Type")
		req.Params["CONTENT_LENGTH"] = r.Header.Get("Content-Length")
		req.Params["HTTPS"] = isHTTPS
		req.Params["GATEWAY_INTERFACE"] = "CGI/1.1"
		req.Params["REMOTE_ADDR"] = remoteAddr
		req.Params["REMOTE_PORT"] = remotePort
		req.Params["SERVER_PORT"] = serverPort
		req.Params["SERVER_NAME"] = r.Host
		req.Params["SERVER_PROTOCOL"] = r.Proto
		req.Params["SERVER_SOFTWARE"] = "gofast"
		req.Params["REDIRECT_STATUS"] = "200"
		req.Params["REQUEST_METHOD"] = r.Method
		req.Params["REQUEST_URI"] = r.RequestURI
		req.Params["QUERY_STRING"] = r.URL.RawQuery
		return
	}

	// ServeHTTP implements http.Handler
	ServeHTTP := func(p *proxy, w http.ResponseWriter, r *http.Request) (errStr string) {
		c, err := gofast.SimpleClientFactory(
			gofast.SimpleConnFactory(p.network, p.address),
			0,
		)()
		if err != nil {
			http.Error(w, "failed to connect to FastCGI application", http.StatusBadGateway)
			log.Printf("gofast: unable to connect to FastCGI application "+
				"(network=%#v, address=%#v, error=%#v)",
				p.network, p.address, err.Error())
			return
		}
		innerCtx, cancel := context.WithCancel(r.Context())
		req := NewRequest(c, r.WithContext(innerCtx))

		// handle the result
		resp, err := c.Do(req)
		cancel() // cancel before reading
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
			errStr = errBuffer.String()
			log.Printf("gofast: error stream from application process "+
				"(network=%#v, address=%#v, error=%#v)",
				p.network, p.address, errStr)
			return
		}

		return
	}

	// create temporary socket in the testing folder
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	sock := dir + "/client.test.sock"

	// create temporary fcgi application server
	// that listens to the socket
	fn := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("accessing FastCGI process")
		fmt.Fprintf(w, "hello world")
	}
	l, err := newApp("unix", sock, fn)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	defer os.Remove(sock)
	defer l.Close()

	// deine a proxy that access the temp fcgi application server
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}

	// test error
	p := &proxy{l.Addr().Network(), l.Addr().String()}
	if want, have := "gofast: timeout or canceled", ServeHTTP(p, w, r); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestClient_StdErr(t *testing.T) {

	// proxy implements Proxy interface
	type proxy struct {
		network string
		address string
	}

	// ServeHTTP implements http.Handler
	ServeHTTP := func(p *proxy, w http.ResponseWriter, r *http.Request) (errStr string) {
		c, err := gofast.SimpleClientFactory(
			gofast.SimpleConnFactory(p.network, p.address),
			0,
		)()
		if err != nil {
			http.Error(w, "failed to connect to FastCGI application", http.StatusBadGateway)
			log.Printf("gofast: unable to connect to FastCGI application "+
				"(network=%#v, address=%#v, error=%#v)",
				p.network, p.address, err.Error())
			return
		}
		req := gofast.NewRequest(c, nil)

		// Some required paramters with invalid values
		req.Params["REQUEST_METHOD"] = ""
		req.Params["SERVER_PROTOCOL"] = ""

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
			errStr = errBuffer.String()
			log.Printf("gofast: error stream from application process "+
				"(network=%#v, address=%#v, error=%#v)",
				p.network, p.address, errStr)
			return
		}

		return
	}

	// create temporary socket in the testing folder
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	sock := dir + "/client.test.sock"

	// create temporary fcgi application server
	// that listens to the socket
	fn := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("accessing FastCGI process")
		fmt.Fprintf(w, "hello world")
	}
	l, err := newApp("unix", sock, fn)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	defer os.Remove(sock)
	defer l.Close()

	// deine a proxy that access the temp fcgi application server
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}

	p := &proxy{l.Addr().Network(), l.Addr().String()}
	if want, have := "cgi: no REQUEST_METHOD in environment", ServeHTTP(p, w, r); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// examine the result
	// FIXME: should show "internal server error"
	if want, have := "", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

}
