package wherr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Wherr struct {
	File    string
	RelPath string
	Line    int
	Message string
	Err     error
}

func Err(location *Location, format string, args ...any) *Wherr {
	message := fmt.Sprintf(format, args...)
	return &Wherr{
		File:    location.File,
		RelPath: location.RelPath,
		Line:    location.Line,
		Message: message,
		Err:     errors.New(message),
	}
}

func Consume(location *Location, err error, prepend string, args ...any) *Wherr {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(prepend, args...) + "\n" + err.Error()
	return &Wherr{
		File:    location.File,
		RelPath: location.RelPath,
		Line:    location.Line,
		Message: message,
		Err:     err,
	}
}

func (e *Wherr) Error() string {
	return fmt.Sprintf("\033[31m[WHERR]\033[0m\033[33m[%s:%d]:\033[0m %s", e.RelPath, e.Line, e.Message)
}

func (e *Wherr) Unwrap() error {
	return e.Err
}

func (e *Wherr) Print() {
	fmt.Fprintf(os.Stderr, "🚨 %s:%d — %s\n", e.RelPath, e.Line, e.Message)
}

func (e *Wherr) Fail() {
	e.Print()
	os.Exit(1)
}

type Location struct {
	RelPath string
	File    string
	Line    int
	Cwd     string
}

func (l Location) Str() string {
	return fmt.Sprintf("Line: %d: File: %s", l.Line, l.RelPath)
}

func Here() *Location {
	_, file, line, _ := runtime.Caller(1)
	cwd, _ := os.Getwd()
	fpath, _ := filepath.Rel(cwd, file)
	return &Location{File: file, RelPath: fpath, Line: line, Cwd: cwd}
}
