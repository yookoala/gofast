package phpfpm_test

import (
	"os"
	"testing"

	"github.com/yookoala/gofast/tools/phpfpm"
)

func TestFindBinary(t *testing.T) {
	paths := phpfpm.ReadPaths(os.Getenv("PATH"))
	bin, err := phpfpm.FindBinary(paths...)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
	if bin == "" {
		t.Logf("Note: expected the testing environment to have php-fpm installed")
		t.Errorf("php-fpm binary not found in your system.")
		t.Logf("Debug - $PATH in env: %#v", paths)
	}
	t.Logf("phpfpm binary found: %s", bin)
}
