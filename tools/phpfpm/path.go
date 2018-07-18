package phpfpm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

// ReadPaths reads a UNIX $PATH (or otherwise formatted alike variable)
// and expand into an array of paths.
func ReadPaths(envPath string) (dirPaths []string) {
	return strings.Split(envPath, ":")
}

// FindBinary finds php-fpm binary in the given paths.
//
// Will return ErrNotExist or other syscall error if
// it have problem finding the file. Will return other
// errors when reading from bad input / directory system.
func FindBinary(dirPaths ...string) (fpmPath string, err error) {
	usualFpmBinPattern := regexp.MustCompile(`^php([\d\.]*)-fpm([\d\.])*$`)
	for _, dirPath := range dirPaths {
		var stat os.FileInfo
		var files []os.FileInfo

		// read the stat of the directory
		stat, err = os.Stat(dirPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // not exists path is simply skipped
			}
			panic(fmt.Sprintf("error reading stat of %#v: %s", dirPath, err))
		}
		if !stat.IsDir() {
			// all existing paths are supposed to be directory
			panic(fmt.Sprintf("not dir: %#v", dirPath))
		}

		// read files inside the directory
		files, err = ioutil.ReadDir(dirPath)
		if err != nil {
			panic(fmt.Sprintf("error reading %#v: %s", dirPath, err))
		}
		for _, file := range files {
			if file.IsDir() {
				// skip directories
				continue
			}
			if usualFpmBinPattern.MatchString(file.Name()) {
				fpmPath = path.Join(dirPath, file.Name())
				return
			}
		}
	}
	err = os.ErrNotExist
	return
}
