package phpfpm_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/yookoala/gophpfpm"

	"github.com/yookoala/gofast/example/phpfpm"
)

var username, phpfpmPath, phpfpmListen string

func init() {
	phpfpmPath = os.Getenv("TEST_PHPFPM_PATH")
	phpfpmListen = os.Getenv("TEST_PHPFPM_LISTEN")
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

func TestHandler(t *testing.T) {

	if phpfpmPath == "" {
		t.Logf("empty TEST_PHPFPM_PATH, skip test")
		return
	}
	if stat, err := os.Stat(phpfpmPath); os.IsNotExist(err) {
		t.Errorf("TEST_PHPFPM_PATH (%#v) not found", phpfpmPath)
		return
	} else if fmode := stat.Mode(); !fmode.IsRegular() {
		t.Errorf("TEST_PHPFPM_PATH (%#v) is not a regular file", phpfpmPath)
		return
	}

	exmpPath := examplePath()
	process := gophpfpm.NewProcess(phpfpmPath)
	process.SetDatadir(path.Join(exmpPath, "var"))
	process.User = username
	process.SaveConfig(path.Join(exmpPath, "etc", "test.handler.conf"))
	if err := process.Start(); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
	defer process.Stop()

	// start the proxy handler
	network, address := process.Address()
	h := phpfpm.NewHandler(
		path.Join(exmpPath, "htdocs"),
		network, address)

	r, err := http.NewRequest("GET", "/index.php", nil)
	if err != nil {
		log.Printf("unexpected error %v", err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	// check results
	if want, have := "hello index", w.Body.String(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}
