package nodejs_test

import (
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/yookoala/gofast/example/nodejs"
)

func examplePath() string {
	basePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(basePath, "src", "index.js")
}

func exampleAssetPath() string {
	basePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(basePath, "assets")
}

func waitConn(socket string) <-chan net.Conn {
	chanConn := make(chan net.Conn)
	go func() {
		log.Printf("wait for socket: %s", socket)
		for {
			if conn, err := net.Dial("unix", socket); err != nil {
				time.Sleep(time.Millisecond * 2)
			} else {
				chanConn <- conn
				break
			}
		}
	}()
	return chanConn
}

func TestHandler(t *testing.T) {
	webapp := examplePath()
	socket := filepath.Join(filepath.Dir(webapp), "test.sock")

	// define webapp.py command
	cmd := exec.Command("node", webapp)
	cmd.Env = append(os.Environ(), "TEST_FCGI_SOCK="+socket)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start the command and wait for its exit
	done := make(chan error, 1)
	go func() {
		if err := cmd.Start(); err != nil {
			done <- err
			return
		}
		// wait if the command started successfully
		log.Printf("started successfully")
		log.Printf("process=%#v", cmd.Process)
		done <- cmd.Wait()
		log.Printf("wait ended")
	}()

	// wait until socket ready
	conn := <-waitConn(socket)
	conn.Close()
	log.Printf("socket ready")

	// start the proxy handler
	h := nodejs.NewHandler(webapp, "unix", socket)

	get := func(path string) (w *httptest.ResponseRecorder, err error) {
		r, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return
	}

	testDone := make(chan bool)
	go func() {
		w, err := get("/")
		if err != nil {
			t.Errorf("unexpected error %v", err)
			testDone <- false
			return
		}
		if want, have := "hello index", w.Body.String(); want != have {
			t.Errorf("expected %#v, got %#v", want, have)
			testDone <- false
			return
		}
		testDone <- true
	}()

	select {
	case testSuccess := <-testDone:
		if !testSuccess {
			log.Printf("test failed")
		}
	case <-time.After(3 * time.Second):
		log.Printf("test timeout")
	case err := <-done:
		if err != nil {
			log.Printf("process done with error = %v", err)
		} else {
			log.Print("process done gracefully without error")
		}
	}

	log.Printf("send SIGTERM")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Fatal("failed to kill: ", err)
	}
	log.Println("process killed")

	os.Remove(socket)
}

func TestMuxHandler(t *testing.T) {
	root := exampleAssetPath() // the "assets" folder
	webapp := examplePath()    // the "src/index.js" file path
	socket := filepath.Join(filepath.Dir(webapp), "test2.sock")

	// define webapp.py command
	cmd := exec.Command("node", webapp)
	cmd.Env = append(os.Environ(), "TEST_FCGI_SOCK="+socket)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start the command and wait for its exit
	done := make(chan error, 1)
	go func() {
		if err := cmd.Start(); err != nil {
			done <- err
			return
		}
		// wait if the command started successfully
		log.Printf("started successfully")
		log.Printf("process=%#v", cmd.Process)
		done <- cmd.Wait()
		log.Printf("wait ended")
	}()

	// wait until socket ready
	conn := <-waitConn(socket)
	conn.Close()
	log.Printf("socket ready")

	// start the proxy handler
	h := nodejs.NewMuxHandler(
		root,
		webapp,
		"unix", socket,
	)

	get := func(path string) (w *httptest.ResponseRecorder, err error) {
		r, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return
	}

	testDone := make(chan bool)
	go func() {

		w, err := get("/responder/")
		if err != nil {
			t.Errorf("unexpected error %v", err)
			testDone <- false
			return
		}
		if want, have := "hello index", w.Body.String(); want != have {
			t.Errorf("expected %#v, got %#v", want, have)
			testDone <- false
			return
		}

		w, err = get("/filter/content.txt")
		if err != nil {
			t.Errorf("unexpected error %v", err)
			testDone <- false
			return
		}
		if want, have := "ereh dlrow olleh", w.Body.String(); want != have {
			t.Errorf("expected %#v, got %#v", want, have)
			testDone <- false
			return
		}
		testDone <- true
	}()

	select {
	case testSuccess := <-testDone:
		if !testSuccess {
			log.Printf("test failed")
		}
	case <-time.After(3 * time.Second):
		log.Printf("test timeout")
	case err := <-done:
		if err != nil {
			log.Printf("process done with error = %v", err)
		} else {
			log.Print("process done gracefully without error")
		}
	}

	log.Printf("send SIGTERM")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Fatal("failed to kill: ", err)
	}
	log.Println("process killed")

	os.Remove(socket)
}

func TestMuxHandler_authorizer(t *testing.T) {
	root := exampleAssetPath() // the "assets" folder
	webapp := examplePath()    // the "src/index.js" file path
	socket := filepath.Join(filepath.Dir(webapp), "test3.sock")

	// define webapp.py command
	cmd := exec.Command("node", webapp)
	cmd.Env = append(os.Environ(), "TEST_FCGI_SOCK="+socket)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// start the command and wait for its exit
	done := make(chan error, 1)
	go func() {
		if err := cmd.Start(); err != nil {
			done <- err
			return
		}
		// wait if the command started successfully
		log.Printf("started successfully")
		log.Printf("process=%#v", cmd.Process)
		done <- cmd.Wait()
		log.Printf("wait ended")
	}()

	// wait until socket ready
	conn := <-waitConn(socket)
	conn.Close()
	log.Printf("socket ready")

	// start the proxy handler
	h := nodejs.NewMuxHandler(
		root,
		webapp,
		"unix", socket,
	)

	testDone := make(chan bool)
	go func() {

		path := "/authorized/responder/"

		r, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return
		}
		w := httptest.NewRecorder()

		// try to access without proper authorization
		h.ServeHTTP(w, r)
		if err != nil {
			t.Errorf("unexpected error %v", err)
			testDone <- false
			return
		}
		if want, have := "authorizer app: permission denied", w.Body.String(); want != have {
			t.Errorf("expected %#v, got %#v", want, have)
			testDone <- false
			return
		}
		if want, have := http.StatusForbidden, w.Code; want != have {
			t.Errorf("expected %#v, got %#v", want, have)
			testDone <- false
			return
		}

		// try to access with proper authorization
		r, err = http.NewRequest("GET", path, nil)
		if err != nil {
			return
		}
		r.Header.Add("Authorization", "hello-auth")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if err != nil {
			t.Errorf("unexpected error %v", err)
			testDone <- false
			return
		}
		if want1, want2, have := "foo: bar!\nhello: howdy!\nhello index", "hello: howdy!\nfoo: bar!\nhello index", w.Body.String(); want1 != have && want2 != have {
			t.Errorf("expected %#v or %#v, got %#v", want1, want2, have)
			testDone <- false
			return
		}
		testDone <- true
	}()

	select {
	case testSuccess := <-testDone:
		if !testSuccess {
			log.Printf("test failed")
		}
	case <-time.After(3 * time.Second):
		log.Printf("test timeout")
	case err := <-done:
		if err != nil {
			log.Printf("process done with error = %v", err)
		} else {
			log.Print("process done gracefully without error")
		}
	}

	log.Printf("send SIGTERM")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Fatal("failed to kill: ", err)
	}
	log.Println("process killed")

	os.Remove(socket)
}
