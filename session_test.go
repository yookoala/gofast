package gofast_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yookoala/gofast"
)

type File struct {
	*strings.Reader
	FileInfo
}

func (f File) Stat() (os.FileInfo, error) {
	return f.FileInfo, nil
}

func (f File) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f File) Close() error {
	return nil
}

type FileEntry struct {

	// stat result
	FileInfo

	// file content
	content string
}

type FileInfo struct {
	// base name of the file
	name string

	// length in bytes for regular files; system-dependent for others
	size int64

	// file mode bits
	mode os.FileMode

	// modification time
	modTime time.Time
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return fi.size
}

func (fi FileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi FileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi FileInfo) Sys() interface{} {
	return nil
}

type VFS map[string]FileEntry

func (vfs VFS) Open(name string) (f http.File, err error) {
	name = strings.Trim(name, "/")
	fi, ok := vfs[name]
	if !ok {
		err = os.ErrNotExist
	}
	f = File{
		FileInfo: fi.FileInfo,
		Reader:   strings.NewReader(fi.content),
	}
	return
}

func TestMapFilterRequest(t *testing.T) {

	dummyModTime := time.Now().Add(-10 * time.Second)

	// dummy filesystem to test with
	vfs := VFS{
		"index.html": FileEntry{
			FileInfo: FileInfo{
				name:    "index.html",
				size:    11,
				mode:    0644,
				modTime: dummyModTime,
			},
			content: "hello world",
		},
	}

	// dummy session handler
	inner := func(client gofast.Client, req *gofast.Request) (resp *gofast.ResponsePipe, err error) {

		if want, have := gofast.RoleFilter, req.Role; want != have {
			t.Errorf("expected: %#v, got: %#v", want, have)
		}

		if req.Data == nil {
			t.Error("filter request requries a data stream")
		} else if content, err := ioutil.ReadAll(req.Data); err != nil {
			t.Errorf("unexpected error: %s", err)
		} else if want, have := "hello world", fmt.Sprintf("%s", content); want != have {
			t.Errorf("expected: %#v, got: %#v", want, have)
		}

		if lastModStr, ok := req.Params["FCGI_DATA_LAST_MOD"]; !ok {
			t.Error("filter request requries param FCGI_DATA_LAST_MOD")
		} else if lastMod, err := strconv.ParseInt(lastModStr, 10, 32); err != nil {
			t.Errorf("invalid parsing FCGI_DATA_LAST_MOD (%s)", err)
		} else if want, have := dummyModTime.Unix(), lastMod; want != have {
			t.Errorf("expected: %#v, got: %#v", want, have)
		}

		if _, ok := req.Params["FCGI_DATA_LENGTH"]; !ok {
			t.Error("filter request requries param FCGI_DATA_LENGTH")
		} else if _, err = strconv.ParseInt(req.Params["FCGI_DATA_LENGTH"], 10, 32); err != nil {
			t.Errorf("invalid parsing FCGI_DATA_LENGTH (%s)", err)
		}
		return
	}
	sess := gofast.MapFilterRequest(vfs)(inner)

	// make dummy request and examine in dummy session handler
	r, err := http.NewRequest("GET", "http://foobar.com/index.html", nil)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	c := gofast.ClientFunc(func(req *gofast.Request) (resp *gofast.ResponsePipe, err error) {
		return
	})
	_, err = sess(c, gofast.NewRequest(r))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
}
