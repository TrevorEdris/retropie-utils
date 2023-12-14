package errors

import "github.com/rotisserie/eris"

var (
	NotImplementedError = eris.New("function not implemented")
)
