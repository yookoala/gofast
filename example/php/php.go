package php

import (
	"net/http"

	"github.com/yookoala/gofast"
)

// NewSimpleHandler returns a fastcgi web server implementation as an http.Handler
// that functions like a traditional file based PHP hosting.
//
// Please note that this handler doesn't handle the fastcgi application process.
// You'd need to start it with other means.
//
// docroot: the document root of the PHP site.
// network: network protocol (tcp / tcp4 / tcp6)
//          or if it is a unix socket, "unix"
// address: IP address and port, or the socket physical address of the fastcgi
//          application.
func NewSimpleHandler(docroot, network, address string) http.Handler {
	connFactory := gofast.SimpleConnFactory(network, address)
	h := gofast.NewHandler(
		gofast.NewPHPFS(docroot)(gofast.BasicSession),
		gofast.SimpleClientFactory(connFactory, 0),
	)
	return h
}

// NewFileEndpointHandler returns a fastcgi web server implementation as an
// http.Handler that referers to a single backend PHP script.
//
// Please note that this handler doesn't handle the fastcgi application process.
// You'd need to start it with other means.
//
// filepath: the path to the endpoint PHP file.
// network:  network protocol (tcp / tcp4 / tcp6)
//           or if it is a unix socket, "unix"
// address:  IP address and port, or the socket physical address of the fastcgi
//           application.
func NewFileEndpointHandler(filepath, network, address string) http.Handler {
	connFactory := gofast.SimpleConnFactory(network, address)
	h := gofast.NewHandler(
		gofast.NewFileEndpoint(filepath)(gofast.BasicSession),
		gofast.SimpleClientFactory(connFactory, 0),
	)
	return h
}
