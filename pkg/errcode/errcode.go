package errcode

import (
	"github.com/gnames/gn"
)

const (
	UnknownError gn.ErrorCode = iota

	CreateDirError
	CopyFileError
	ReadFileError

	CreateLogFileError
)
