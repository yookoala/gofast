package gofast_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yookoala/gofast"
)

func TestHandler(t *testing.T) {

	// create temporary socket in the testing folder
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	sock := dir + "/test.handler.sock"

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
	p := gofast.NewHandler(
		gofast.NewPHPFS(""),
		gofast.SimpleClientFactory(
			gofast.SimpleConnFactory(
				l.Addr().Network(),
				l.Addr().String(),
			),
			0,
		),
	)
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
