package gofast

// Request hold information of a standard
// FastCGI request
type Request interface {
	// GetID returns request ID
	GetID() uint16
}

// request is the default implementation of Request
type request struct {
	reqID     uint16
	params    map[string]string
	buf       [1024]byte
	rawParams []byte
	keepConn  bool
}

// GetID implements Request interface
func (r *request) GetID() uint16 {
	return r.reqID
}
