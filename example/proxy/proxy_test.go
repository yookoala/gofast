package proxy_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"testing"

	"github.com/yookoala/gofast/example/proxy"
)

func app() (l net.Listener, err error) {
	l, err = net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		return
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
	}
	go fcgi.Serve(l, http.HandlerFunc(handler))
	return
}

func TestProxy(t *testing.T) {
	l, err := app()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	p := proxy.New(l.Addr().String())
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	p.ServeHTTP(w, r)

	if want, have := "", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}
