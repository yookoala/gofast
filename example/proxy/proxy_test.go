package proxy_test

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yookoala/gofast/example/proxy"
)

func app(network, address string) (l net.Listener, err error) {
	l, err = net.Listen(network, address)
	if err != nil {
		return
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("in app")
		fmt.Fprintf(w, "hello world")
	}
	go fcgi.Serve(l, http.HandlerFunc(handler))
	return
}

func TestProxy(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	sock := dir + "/test.sock"
	log.Printf(sock)

	l, err := app("unix", sock)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	defer os.Remove(sock)
	defer l.Close()

	p := proxy.New(l.Addr().Network(), l.Addr().String())
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	p.ServeHTTP(w, r)

	if notWant, have := "", w.Body.String(); notWant == have {
		t.Errorf("not expected %#v, got %#v", notWant, have)
	}
}
