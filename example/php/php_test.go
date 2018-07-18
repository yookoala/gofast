package php_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-restit/lzjson"

	"github.com/yookoala/gofast/example/php"
	"github.com/yookoala/gofast/tools/phpfpm"
)

var username, phpfpmPath string

func init() {
	var err error

	// defined in environment
	if phpfpmPath = os.Getenv("TEST_PHPFPM_PATH"); phpfpmPath != "" {
		// do nothing
	} else if phpfpmPath, err = phpfpm.FindBinary(phpfpm.ReadPaths(os.Getenv("PATH"))...); err != nil {
		panic(err)
	}
	username = os.Getenv("USER")
}

func isExamplePath(testPath string) bool {
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(path.Join(testPath, "var")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(path.Join(testPath, "etc")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(path.Join(testPath, "htdocs")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(path.Join(testPath, "htdocs", "index.php")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(path.Join(testPath, "htdocs", "form.php")); os.IsNotExist(err) {
		return false
	}
	return true
}

func examplePath() string {
	basePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if isExamplePath(basePath) {
		return basePath
	}

	basePath = path.Join(basePath, "example", "phpfpm")
	if isExamplePath(basePath) {
		return basePath
	}

	panic("example path not found")
}

func get(h http.Handler, path string) (w *httptest.ResponseRecorder, err error) {
	r, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return
}

func post(h http.Handler, path string, payload string) (w *httptest.ResponseRecorder, err error) {
	var reader io.Reader
	reader = strings.NewReader(payload)
	r, err := http.NewRequest("POST", path, reader)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", fmt.Sprintf("%d", len(payload)))
	if err != nil {
		return
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return
}

func initEnv(t *testing.T, name string) (exmpPath string, process *phpfpm.Process) {
	if phpfpmPath == "" {
		t.Skip("empty TEST_PHPFPM_PATH, skip test")
	}
	if stat, err := os.Stat(phpfpmPath); os.IsNotExist(err) {
		t.Errorf("TEST_PHPFPM_PATH (%#v) not found", phpfpmPath)
		return
	} else if fmode := stat.Mode(); !fmode.IsRegular() {
		t.Errorf("TEST_PHPFPM_PATH (%#v) is not a regular file", phpfpmPath)
		return
	}

	exmpPath = examplePath()
	process = phpfpm.NewProcess(phpfpmPath)
	process.SetName(name)
	process.SetDatadir(path.Join(exmpPath, "var"))
	process.User = username
	process.SaveConfig(path.Join(exmpPath, "etc", "test.handler.conf"))
	if err := process.Start(); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}

	return
}
func TestNewSimpleHandler(t *testing.T) {

	exmpPath, process := initEnv(t, "phpfpm1")
	defer process.Stop()

	// start the proxy handler
	network, address := process.Address()
	h := php.NewSimpleHandler(
		path.Join(exmpPath, "htdocs"),
		network, address)

	// check results
	w, err := get(h, "/")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "hello index", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
	if want, have := "World", w.Header().Get("X-Hello"); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
	if want, have := "Bar", w.Header().Get("X-Foo"); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	w, err = get(h, "/index.php")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "hello index", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	w, err = get(h, "/form.php")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	formPrefix := "<!DOCTYPE html>\n<html>\n<head>\n  <title>Simple Form"
	if have := w.Body.String(); !strings.HasPrefix(have, formPrefix) {
		t.Errorf("expected to start with %#v, got %#v", formPrefix, have)
	}

	w, err = get(h, "/form.php?hello=world")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "$_GET = array (\n  'hello' => 'world',\n)", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	form := url.Values{}
	form.Add("text_input", "hello world")
	w, err = post(h, "/form.php", form.Encode())
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "$_POST = array (\n  'text_input' => 'hello world',\n)", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestNewSimpleHandler__ErrorStream(t *testing.T) {

	exmpPath, process := initEnv(t, "phpfpm2")
	defer process.Stop()

	// start the proxy handler
	network, address := process.Address()
	h := php.NewSimpleHandler(
		path.Join(exmpPath, "htdocs"),
		network, address)

	// check results
	w, err := get(h, "/error.php")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "1. some standard output.\n3. some more standard output.\n5. unparsed.\n", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// check results
	w, err = get(h, "/error.php?error_only=1")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	if want, have := "", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestNewFileEndpointHandler(t *testing.T) {

	exmpPath, process := initEnv(t, "phpfpm3")
	defer process.Stop()

	// start the proxy handler
	var w *httptest.ResponseRecorder
	var err error
	var resp lzjson.Node
	network, address := process.Address()
	h := php.NewFileEndpointHandler(
		path.Join(exmpPath, "htdocs", "vars.php"),
		network, address)

	// check results for a proper path
	w, err = get(h, "/")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	resp = lzjson.Decode(w.Body)
	if resp.Get("$_SERVER").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if resp.Get("$_SERVER").Get("REQUEST_URI").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER.REQUEST_URI not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if want, have := "/", resp.Get("$_SERVER").Get("REQUEST_URI").String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// check results for a proper path
	w, err = get(h, "/hello/world")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	resp = lzjson.Decode(w.Body)
	if resp.Get("$_SERVER").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if resp.Get("$_SERVER").Get("REQUEST_URI").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER.REQUEST_URI not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if want, have := "/hello/world", resp.Get("$_SERVER").Get("REQUEST_URI").String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// check results for a proper path
	w, err = get(h, "/index.php")
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	resp = lzjson.Decode(w.Body)
	if resp.Get("$_SERVER").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if resp.Get("$_SERVER").Get("REQUEST_URI").Type() == lzjson.TypeUndefined {
		t.Errorf("$_SERVER.REQUEST_URI not set in response json")
		t.Logf("Response JSON: %s", resp.Raw())
	} else if want, have := "/index.php", resp.Get("$_SERVER").Get("REQUEST_URI").String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}
