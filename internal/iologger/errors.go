package iologger

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

func CreateLogFileError(path string, err error) error {
	msg := "Cannot create log file <em>%s</em>"
	vars := []any{path}
	return &gn.Error{
		Code: errcode.CreateLogFileError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cannot create log file: %w", err),
	}
}
