package phpfpm_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/go-ini/ini"

	"github.com/yookoala/gofast/example/phpfpm"
)

var phpfpmPath, phpfpmListen string

func init() {
	phpfpmPath = os.Getenv("TEST_PHPFPM_PATH")
	phpfpmListen = os.Getenv("TEST_PHPFPM_LISTEN")
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

func genConfig(basePath string) (confPath, network, address string) {

	if basePath == "" {
		panic("empty basePath")
	}

	// treat as tcp by default
	network = "tcp"
	address = phpfpmListen
	listen := phpfpmListen

	// check if use socket
	if listen == "" {
		network = "unix"
		address = path.Join(basePath, "var", "phpfpm.sock")
		listen = address
	} else if reSocket := regexp.MustCompile("^(unix)\\:(.*)$"); reSocket.MatchString(listen) {
		network = "unix"
		address = reSocket.FindStringSubmatch(phpfpmListen)[1]
		listen = address
	}

	confPath = path.Join(basePath, "etc", "phpfpm.conf")
	pidfile := path.Join(basePath, "var", "phpfpm.pid")

	cfg := ini.Empty()
	cfg.NewSection("global")
	cfg.Section("global").NewKey("pid", pidfile)
	cfg.Section("global").NewKey("error_log", path.Join(basePath, "var", "phpfpm.error_log"))
	cfg.NewSection("www")
	cfg.Section("www").NewKey("listen", listen)
	cfg.Section("www").NewKey("pm", "dynamic")
	cfg.Section("www").NewKey("pm.max_children", "5")
	cfg.Section("www").NewKey("pm.start_servers", "2")
	cfg.Section("www").NewKey("pm.min_spare_servers", "1")
	cfg.Section("www").NewKey("pm.max_spare_servers", "3")

	cfg.SaveTo(confPath)
	return
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
	confPath, network, address := genConfig(exmpPath)
	cmd := &exec.Cmd{
		Path: phpfpmPath,
		Args: append([]string{phpfpmPath},
			"--fpm-config", confPath,
			"-F",  // start foreground
			"-n",  // no ini file
			"-e"), // extended information
		Stderr: os.Stderr, // for now
	}

	stdoutRead, err := cmd.StdoutPipe()
	if err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}
	defer stdoutRead.Close()

	// start the command then wait to kill after test end
	if err := cmd.Start(); err != nil {
		t.Errorf("unexpected error %v", err)
		return
	}

	// sleep a bit and wait for the process start
	time.Sleep(time.Millisecond * 500)
	defer cmd.Process.Wait()
	defer cmd.Process.Signal(os.Interrupt)

	t.Logf("started php-fpm on process %#v", cmd.Process.Pid)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Logf("process.Signal on pid %d returned: %v\n", cmd.Process.Pid, err)
	}

	// start the proxy handler
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
