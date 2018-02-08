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

// NewFilterHandler for advanced test for muxing
// which will pass a requested file, in root folder, to
// the fastcgi application for filtering
func NewFilterHandler(root string, clientFactory gofast.ClientFactory) http.Handler {
	return gofast.NewHandler(
		gofast.NewFilterLocalFS(root)(gofast.BasicSession),
		clientFactory,
	)
}

// NewResponderHandler for advanced test for muxing
func NewResponderHandler(entrypoint string, clientFactory gofast.ClientFactory) http.Handler {
	return gofast.NewHandler(
		gofast.NewFileEndpoint(entrypoint)(gofast.BasicSession),
		clientFactory,
	)
}

// NewMuxHandler create advanced muxing example
func NewMuxHandler(
	root string, // root folder for filter data
	entrypoint string, // entrypoint for building params to responder
	network, address string,
) http.Handler {

	// common client pool for both filter and responder handler
	connFactory := gofast.SimpleConnFactory(network, address)
	pool := gofast.NewClientPool(
		gofast.SimpleClientFactory(connFactory, 0),
		10,
		60*time.Second,
	)

	// mux filter and responder in different folder
	mux := http.NewServeMux()
	mux.Handle("/filter/", http.StripPrefix("/filter/", NewFilterHandler(
		root,
		pool.CreateClient,
	)))
	mux.Handle("/responder/", http.StripPrefix("/responder/", NewResponderHandler(
		entrypoint,
		pool.CreateClient,
	)))

	// authorized endpoint
	authorizer := gofast.NewAuthorizer(
		pool.CreateClient,
		gofast.NewAuthPrepare()(gofast.BasicSession),
	)
	// this endpoint is guarded by authorizer application
	//
	// note: you may use different pool / connFactory to connect
	// to different fastcgi authorizer application than the inner
	// content.
	mux.Handle(
		"/authorized/responder/",
		http.StripPrefix("/authorized/responder/", authorizer.Wrap(NewResponderHandler(
			entrypoint,
			pool.CreateClient,
		))),
	)
	return mux
}
