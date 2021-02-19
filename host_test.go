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
		w.Header().Add("X-Hello", "World 1")
		w.Header().Add("X-Hello", "World 2")
		w.Header().Add("X-Foo", "Bar 1")
		w.Header().Add("X-Foo", "Bar 2")
		w.WriteHeader(201)
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
		gofast.NewPHPFS("")(gofast.BasicSession),
		gofast.SimpleClientFactory(
			gofast.SimpleConnFactory(
				l.Addr().Network(),
				l.Addr().String(),
			),
		),
	)
	w := httptest.NewRecorder()

	// request the application server
	r, err := http.NewRequest("GET", "/add", nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
	p.ServeHTTP(w, r)

	// examine the body
	if want, have := "hello world", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// examine the code
	if want, have := 201, w.Code; want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// examine the header
	if field, ok := w.HeaderMap["X-Hello"]; ok {
		for i, v := range field {
			if want, have := fmt.Sprintf("World %d", i+1), v; want != have {
				t.Errorf("expected %#v, got %#v", want, have)
			}
		}
	}
	if field, ok := w.HeaderMap["X-Foo"]; ok {
		for i, v := range field {
			if want, have := fmt.Sprintf("Bar %d", i+1), v; want != have {
				t.Errorf("expected %#v, got %#v", want, have)
			}
		}
	}
}
