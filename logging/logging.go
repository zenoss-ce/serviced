package logging

import (
	"runtime"
	"strings"

	"github.com/zenoss/logri"
	"fmt"
	"github.com/Sirupsen/logrus"
	"os"
)

func init() {
	logri.AddHook(ContextHook{})
}

func pkgFromFunc(funcname string) string {
	subpkg := strings.TrimPrefix(funcname, prefix)
	parts := strings.Split(subpkg, ".")
	pkg := ""
	if parts[len(parts)-2] == "(" {
		pkg = strings.Join(parts[0:len(parts)-2], ".")
	} else {
		pkg = strings.Join(parts[0:len(parts)-1], ".")
	}
	return strings.Replace(pkg, "/", ".", -1)
}

// PackageLogger returns a logger for a given package.
func PackageLogger_old() *logri.Logger {
	pc := make([]uintptr, 3, 3)
	count := runtime.Callers(2, pc)
	for i := 0; i < count; i++ {
		fu := runtime.FuncForPC(pc[i])
		name := fu.Name()
		if strings.Contains(name, prefix) {
			return logri.GetLogger(pkgFromFunc(name))
		}
	}
	return logri.GetLogger("")
}

func PackageLogger() *logri.Logger {
	pc := make([]uintptr, 3, 3)
	_ = runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc)
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		name := frame.Func.Name()
		if strings.Contains(name, prefix) {
			return logri.GetLogger(pkgFromFunc(name))
		}
	}
	return logri.GetLogger("")
}

func AuditLogger() *logri.Logger {
	al := logri.GetLogger(audit)
	al.SetLevel(logrus.InfoLevel, false)
	fileopt := map[string]string {"file": auditlogloc}
	w, err := logri.GetOutputWriter(logri.FileOutput, fileopt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting output writer for %s: %s\n", auditlogloc, err)
		return nil
	}
	al.SetOutput(w)
	return al
}