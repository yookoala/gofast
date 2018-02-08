package gofast

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
)

// NewAuthRequest returns a new *http.Request
// and a *Request with the body buffered
// into new NopReader.
func NewAuthRequest(orgl *http.Request) (r *http.Request, req *Request, err error) {

	// new request struct that inherits orgl values
	r = &http.Request{}
	*r = *orgl

	var stdin io.ReadCloser

	// clone the raw request content into r.Body and stdin
	// if there is any body
	if orgl.Body != nil {
		var content []byte
		content, err = ioutil.ReadAll(orgl.Body)
		if err != nil {
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(content))
		stdin = ioutil.NopCloser(bytes.NewBuffer(content))
	}

	// generate the request
	req = &Request{
		Raw:    orgl,
		Role:   RoleAuthorizer,
		Params: make(map[string]string),
		Stdin:  stdin,
		Data:   nil,
	}
	return
}

// NewAuthorizer creates an authorizer
func NewAuthorizer(clientFactory ClientFactory, sessionHandler SessionHandler) *Authorizer {
	return &Authorizer{
		clientFactory,
		sessionHandler,
	}
}

// Authorizer guard a given http.Handler
//
// Since this is implemented as a generic http.Handler middleware,
// you may use this with other non-gofast library components as
// long as they implements the http.Handler interface.
type Authorizer struct {
	clientFactory  ClientFactory
	sessionHandler SessionHandler
}

// Wrap method is a generic http.Handler middleware. Requests
// to wrapped hander would go through the fastcgi authorizer
// first. If not authorized, the request will not reach wrapped
// hander.
func (ar Authorizer) Wrap(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// generate auth request
		innerReq, req, err := NewAuthRequest(r)
		if err != nil {
			w.Header().Add("Content-Type", "text/html; charset=utf8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "%s", err)
			return
		}

		// get client to fastcgi application
		c, err := ar.clientFactory()
		if err != nil {
			w.Header().Add("Content-Type", "text/html; charset=utf8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to connect to authorizer: %s", err)
			return
		}

		// make request with client
		resp, err := ar.sessionHandler(c, req)
		if err != nil {
			w.Header().Add("Content-Type", "text/html; charset=utf8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error with authorizer request: %s", err)
			return
		}

		ew := new(bytes.Buffer)
		rw := httptest.NewRecorder() // FIXME: should do this without httptest
		resp.WriteTo(rw, ew)

		// if code is not http.StatusOK (200)
		if rw.Code != http.StatusOK {
			// copy header map
			for k, m := range rw.Header() {
				for _, v := range m {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(rw.Code)
			fmt.Fprintf(w, rw.Body.String())
			return
		}

		// if error stream is not empty
		if ew.Len() > 0 {
			w.Header().Add("Content-Type", "text/html; charset=utf8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error reading authorizer response: %s", err)
			log.Printf("gofast: error stream from application process %s",
				ew.String())
			return
		}

		// no problem from authorizer
		// pass down the inner handler
		inner.ServeHTTP(w, innerReq)
	})
}
