package util

import (
	"os"
)

func Errno(err os.Error) (errno os.Errno, ok bool) {
	if e, ok := err.(*os.PathError); ok {
		err = e.Error
	}
	errno, ok = err.(os.Errno)
	return
}
