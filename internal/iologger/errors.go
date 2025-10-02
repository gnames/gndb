package iologger

import (
	"fmt"
	"runtime"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

func CreateLogFileError(path string, err error) error {
	msg := "Cannot create log file <em>%s</em>"
	vars := []any{path}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	return &gn.Error{
		Code: errcode.CreateLogFileError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: cannot create log file: %w",
			fn, err),
	}
}
