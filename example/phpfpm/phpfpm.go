package phpfpm

import (
	"net/http"
	"path"

	"github.com/yookoala/gofast"
)

// NewHandler returns a new FastCGI handler
func NewHandler(root, network, address string) http.Handler {
	h := gofast.NewHandler(network, address)
	h.SetBeforeDo(func(req *gofast.Request, r *http.Request) (out *gofast.Request, err error) {
		out = req
		out.Params["SCRIPT_FILENAME"] = path.Join(root, r.URL.Path)
		return
	})
	return h
}
