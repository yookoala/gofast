package proxy_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yookoala/gofast/example/proxy"
)

func app(network, address string, fn http.HandlerFunc) (l net.Listener, err error) {
	l, err = net.Listen(network, address)
	if err != nil {
		return
	}
	go fcgi.Serve(l, http.HandlerFunc(fn))
	return
}

func TestProxy(t *testing.T) {

	// create temporary socket in the testing folder
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	sock := dir + "/test.sock"

	// create temporary fcgi application server
	// that listens to the socket
	fn := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("accessing FastCGI process")
		fmt.Fprintf(w, "hello world")
	}
	l, err := app("unix", sock, fn)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	defer os.Remove(sock)
	defer l.Close()

	// deine a proxy that access the temp fcgi application server
	p := proxy.New(l.Addr().Network(), l.Addr().String())
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	p.ServeHTTP(w, r)

	// examine the result
	if want, have := "hello world", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}
