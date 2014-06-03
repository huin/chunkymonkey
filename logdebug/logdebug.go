package logdebug

import (
	"log"
	"runtime"
)

func LogStack(format string, v ...interface{}) {
	log.Printf(format, v...)
	pcs := make([]uintptr, 256)
	n := runtime.Callers(2, pcs)
	for i := 0; i < n; i++ {
		pc := pcs[i]
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			log.Printf("  at %s:%d in %s", file, line, fn.Name())
		} else {
			log.Printf("  at %x in <unknown>", pc)
		}
	}
}
