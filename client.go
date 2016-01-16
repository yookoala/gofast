package gofast

type client struct {
	pass string
}

// Client is a client interface of FastCGI
// application process through given
// host:port / socket definition
type Client interface {
}

// NewClient returns a Client of the given
// pass (host:port or socket)
func NewClient(pass string) Client {
	return &client{
		pass: pass,
	}
}
