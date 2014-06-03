package util

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
)

// OpenFileUniqueName creates a file with a unique (and randomly generated)
// filename with the given path and name prefix. It is opened with
// flag|os.O_CREATE|os.O_EXCL; os.O_WRONLY or os.RDWR should be specified for
// flag at minimum. It is the caller's responsibility to close (and maybe
// delete) the file when they have finished using it.
// TODO: Use a function from the standard library for doing this, if available
func OpenFileUniqueName(prefix string, flag int, perm os.FileMode) (file *os.File, err error) {
	useFlag := flag | os.O_CREATE | os.O_EXCL
	for i := 0; i < 1000; i++ {
		rnd := rand.Int()
		if file, err := os.OpenFile(prefix+strconv.Itoa(rnd), useFlag, perm); err == nil {
			return file, err
		} else if os.IsExist(err) {
			// Try again up to 1000 times.
			continue
		} else {
			return nil, err
		}
	}
	return nil, errors.New("gave up trying to create unique filename")
}
