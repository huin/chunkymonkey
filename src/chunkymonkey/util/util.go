package util

import (
	"os"
	"math/rand"
	"strconv"
)

func Errno(err error) (errno os.Errno, ok bool) {
	if e, ok := err.(*os.PathError); ok {
		err = e.Error
	}
	errno, ok = err.(os.Errno)
	return
}

// OpenFileUniqueName creates a file with a unique (and randomly generated)
// filename with the given path and name prefix. It is opened with
// flag|os.O_CREATE|os.O_EXCL; os.O_WRONLY or os.RDWR should be specified for
// flag at minimum. It is the caller's responsibility to close (and maybe
// delete) the file when they have finished using it.
func OpenFileUniqueName(prefix string, flag int, perm uint32) (file *os.File, err error) {
	useFlag := flag | os.O_CREATE | os.O_EXCL
	for i := 0; i < 1000; i++ {
		rnd := rand.Int63()
		if file, err := os.OpenFile(prefix+strconv.Itob64(rnd, 16), useFlag, perm); err == nil {
			return file, err
		} else {
			if errno, ok := Errno(err); ok && errno == os.EEXIST {
				continue
			}
			return nil, err
		}
	}
	return nil, os.NewError("gave up trying to create unique filename")
}
