package errors

import "github.com/rotisserie/eris"

var (
	NotImplementedError  = eris.New("function not implemented")
	DefaultUsernameError = eris.New("must not use default username")
	InvalidUsernameError = eris.New("invalid username")
)

func NewInvalidUsernameWithReasonError(reason string) error {
	return eris.Wrap(InvalidUsernameError, reason)
}
