package gofast

// Request is an interface of a standard
// FastCGI request
type Request struct {
}

// NewRequest returns a standard FastCGI request
func NewRequest() *Request {
	return &Request{}
}
