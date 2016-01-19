package phpfpm

import (
	"net/http"
	"path"
	"strings"

	"github.com/yookoala/gofast"
)

// NewHandler returns a new FastCGI handler
func NewHandler(root, network, address string) http.Handler {
	h := gofast.NewHandler(network, address)
	h.SetBeforeDo(func(req *gofast.Request, r *http.Request) (out *gofast.Request, err error) {
		out = req
		urlPath := r.URL.Path
		if strings.HasSuffix(urlPath, "/") {
			urlPath = path.Join(urlPath, "index.php") // directory index
		}
		out.Params["SCRIPT_FILENAME"] = path.Join(root, urlPath)
		return
	})
	return h
}
