package iofs

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

func CreateDirError(dir string, err error) error {
	msg := "Cannot create %s"
	vars := []any{dir}
	return &gn.Error{
		Code: errcode.CreateDirError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cannot create directory: %w", err),
	}
}

func CopyFileError(file string, err error) error {
	msg := "Cannot copy config file to %s"
	vars := []any{file}
	return &gn.Error{
		Code: errcode.CopyFileError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cannot copy file: %w", err),
	}
}

func ReadFileError(path string, err error) error {
	msg := "Cannot read <em>%s</em>"
	vars := []any{path}
	return &gn.Error{
		Code: errcode.ReadFileError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cannot read %s: %w", path, err),
	}
}
