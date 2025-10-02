package iofs

import (
	"fmt"
	"runtime"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

func CreateDirError(dir string, err error) error {
	msg := "Cannot create %s"
	vars := []any{dir}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	return &gn.Error{
		Code: errcode.CreateDirError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: cannot create directory: %w",
			fn, err),
	}
}

func CopyFileError(file string, err error) error {
	msg := "Cannot copy config file to %s"
	vars := []any{file}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	return &gn.Error{
		Code: errcode.CopyFileError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: cannot copy file: %w",
			fn, err),
	}
}

func ReadFileError(path string, err error) error {
	msg := "Cannot read <em>%s</em>"
	vars := []any{path}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	return &gn.Error{
		Code: errcode.ReadFileError,
		Err:  fmt.Errorf("from %s: cannot read %s: %w", fn, path, err),
		Msg:  msg,
		Vars: vars,
	}
}
