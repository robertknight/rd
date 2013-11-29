package procinfo

import "errors"

type Proc struct {
	Id         int
	ExePath    string
	CurrentDir string
}

var ErrAccess = errors.New("Unable to read process info")
