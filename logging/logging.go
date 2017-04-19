package logging

import (
	"runtime"
	"strings"

	"github.com/zenoss/logri"
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
	// TODO: if ServicedLogDir() doesn't exist, this won't work - logri doesn't create path.
	//logfn := path.Join(ServicedLogDir(), auditlogname)
	//ofd := map[string]string{"file": logfn}
	//ow, err := logri.GetOutputWriter(logri.FileOutput, ofd)
	//if err != nil {
	//	al.WithError(err).Info("Unable to get output writer for audit log.")
	//	return al
	//}
	//al.SetOutput(ow)
	return al
}

// ServicedLogDir gets the serviced log directory
func ServicedLogDir() string {
	return "/var/log/serviced"
}
