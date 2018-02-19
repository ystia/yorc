package hostspool

import (
	"github.com/pkg/errors"
)

type hostNotFoundError struct{}

func (e hostNotFoundError) Error() string {
	return "host not found in pool"
}

// IsHostNotFoundError checks if an error is an "host not found" error
func IsHostNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(hostNotFoundError)
	return ok
}

type hostAlreadyExistError struct{}

func (e hostAlreadyExistError) Error() string {
	return "a host with the same name already exists in the pool"
}

// IsHostAlreadyExistError checks if an error is an "host already exists" error
func IsHostAlreadyExistError(err error) bool {
	_, ok := errors.Cause(err).(hostAlreadyExistError)
	return ok
}

type badRequestError struct {
	msg string
}

func (e badRequestError) Error() string {
	return e.msg
}

// IsBadRequestError checks if an error is an error due to a bad input
func IsBadRequestError(err error) bool {
	_, ok := errors.Cause(err).(badRequestError)
	return ok
}

type noMatchingHostFoundError struct{}

func (e noMatchingHostFoundError) Error() string {
	return "no matching host found"
}

// IsNoMatchingHostFoundError checks if an error is an error due to no hosts match the given filters if any
func IsNoMatchingHostFoundError(err error) bool {
	_, ok := errors.Cause(err).(noMatchingHostFoundError)
	return ok
}
