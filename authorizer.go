package gofast

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// NewAuthRequest returns a new *http.Request
// and a *Request with the body buffered
// into new NopReader.
func NewAuthRequest(id uint16, orgl *http.Request) (r *http.Request, req *Request, err error) {

	// new request struct that inherits orgl values
	r = &http.Request{}
	*r = *orgl

	// clone the raw request content into r.Body and stdin
	content, err := ioutil.ReadAll(orgl.Body)
	if err != nil {
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(content))
	stdin := ioutil.NopCloser(bytes.NewBuffer(content))

	// generate the request
	req = &Request{
		Raw:    orgl,
		Role:   RoleAuthorizer,
		ID:     id,
		Params: make(map[string]string),
		Stdin:  stdin,
		Data:   nil,
	}
	return
}
