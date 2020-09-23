package gofast_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// appServer implements Proxy interface
type appServer struct {
	listener net.Listener
	sock     string
}

func (p *appServer) Network() string {
	return p.listener.Addr().Network()
}

func (p *appServer) Address() string {
	return p.listener.Addr().String()
}

func (p *appServer) Close() {
	os.Remove(p.sock)
	p.listener.Close()
}

func newAppServer(sockName string, fn http.HandlerFunc) (p *appServer, err error) {
	// create temporary socket in the testing folder
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	sock := filepath.Join(dir, sockName)

	// create temporary fcgi application server
	// that listens to the socket
	l, err := newApp("unix", sock, fn)
	if err != nil {
		return
	}

	p = &appServer{
		listener: l,
		sock:     sock,
	}
	return
}

func testHandlerForCancel(t *testing.T, p *appServer, w http.ResponseWriter, r *http.Request) (errStr string) {

	NewRequest := func(r *http.Request) (req *gofast.Request) {
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

		req = gofast.NewRequest(r)
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

	c, err := gofast.SimpleClientFactory(
		gofast.SimpleConnFactory(p.Network(), p.Address()),
		0,
	)()
	if err != nil {
		http.Error(w, "failed to connect to FastCGI application", http.StatusBadGateway)
		t.Logf("web server: unable to connect to FastCGI application "+
			"(network=%#v, address=%#v, error=%#v)",
			p.Network(), p.Address(), err.Error())
		return
	}
	innerCtx, cancel := context.WithCancel(r.Context())
	req := NewRequest(r.WithContext(innerCtx))

	// cancel before request
	cancel()

	// wait for the cancel signal to kick in
	// or the artificial wait timeout
	select {
	case <-innerCtx.Done():
		t.Logf("cancel effective")
	case <-time.After(100 * time.Millisecond):
		t.Logf("time out reach")
	}

	// handle the result
	resp, err := c.Do(req)
	if err != nil {
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		t.Logf("web server: unable to process request "+
			"(network=%#v, address=%#v, error=%#v)",
			p.Network(), p.Address(), err.Error())
	}

	errBuffer := new(bytes.Buffer)
	resp.WriteTo(w, errBuffer)

	if errBuffer.Len() > 0 {
		errStr = errBuffer.String()
		t.Logf("web server: error stream from application process "+
			"(network=%#v, address=%#v, error=%#v)",
			p.Network(), p.Address(), errStr)
		return
	}

	return
}

func TestClient_canceled(t *testing.T) {

	// create a temp dummy fastcgi application server
	p, err := newAppServer("client.test.sock", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("accessing FastCGI process")
		time.Sleep(10 * time.Second) // mimic long running process
		fmt.Fprintf(w, "hello world")
		t.Logf("FastCGI process finished")
	})
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	defer p.Close()

	// deine a appServer that access the temp fcgi application server
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}

	// test response
	if want, have := "", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// test error
	if want, have := "gofast: timeout or canceled", testHandlerForCancel(t, p, w, r); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestClient_StdErr(t *testing.T) {

	// create a temp dummy fastcgi application server
	p, err := newAppServer("client.test.sock", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("accessing FastCGI process")
		fmt.Fprintf(w, "hello world")
	})
	defer p.Close()

	// Do the actual request
	doRequest := func(w http.ResponseWriter, r *http.Request) (errStr string) {
		c, err := gofast.SimpleClientFactory(
			gofast.SimpleConnFactory(p.Network(), p.Address()),
			0,
		)()
		if err != nil {
			errStr = "web server: unable to connect to FastCGI application: " + err.Error()
			return
		}
		req := gofast.NewRequest(nil)

		// Some required parameters with invalid values
		req.Params["REQUEST_METHOD"] = ""
		req.Params["SERVER_PROTOCOL"] = ""

		// handle the result
		resp, err := c.Do(req)
		if err != nil {
			errStr = "web server: unable to connect to process request: " + err.Error()
			return
		}
		errBuffer := new(bytes.Buffer)
		resp.WriteTo(w, errBuffer)

		if errBuffer.Len() > 0 {
			// direct return the error stream
			errStr = errBuffer.String()
			return
		}
		return
	}

	// define an appServer that access the temp fcgi application server
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}

	if want, have := "cgi: no REQUEST_METHOD in environment", doRequest(w, r); want != have {
		t.Logf("network=%#v, address=%#v", p.Network(), p.Address())
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// examine the result
	// FIXME: should show "internal server error"
	if want, have := "", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}
