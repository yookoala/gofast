package nodejs

import (
	"net/http"
	"time"

	"github.com/yookoala/gofast"
)

// NewHandler returns a fastcgi web server implementation as an http.Handler
// Please note that this handler doesn't handle the fastcgi application process.
// You'd need to start it with other means.
//
// entrypoint: the full path to the application entrypoint file (e.g. webapp.py)
//             or equivlant path for fastcgi application to identify itself.
// network: network protocol (tcp / tcp4 / tcp6)
//          or if it is a unix socket, "unix"
// address: IP address and port, or the socket physical address of the fastcgi
//          application.
func NewHandler(entrypoint, network, address string) http.Handler {
	connFactory := gofast.SimpleConnFactory(network, address)
	pool := gofast.NewClientPool(
		gofast.SimpleClientFactory(connFactory, 0),
		10,
		60*time.Second,
	)
	h := gofast.NewHandler(
		gofast.NewFileEndpoint(entrypoint)(gofast.BasicSession),
		pool.CreateClient,
	)
	return h
}
