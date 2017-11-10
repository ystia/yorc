package plugin

import (
	"fmt"
	"github.com/pkg/errors"
)

// RPCError is a wrapper dedicated to allow passing errors via RPC encoding
// It must be used for RPC server side in the response call method (ie plugin side actually)
type RPCError struct {
	Message string
	Stack   string
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// This allows RPCError to implement error and display message plus stack
func (pErr *RPCError) Error() string {
	return fmt.Sprintf("%s%s", pErr.Message, pErr.Stack)
}

// RPCError needs to be explicitly typed "nil" before casting to error
// https://golang.org/doc/faq#nil_error
func toError(pErr *RPCError) error {
	if pErr != nil {
		return pErr
	}

	return nil
}

// NewRPCError allows to instantiate a RPCError from an error builtin type
func NewRPCError(err error) *RPCError {
	return &RPCError{Message: err.Error(), Stack: getStackTrace(err)}
}

// NewRPCErrorFromMessage allows to instantiate a RPCError from a message and variadic arguments
func NewRPCErrorFromMessage(m string, args ...interface{}) *RPCError {
	return &RPCError{Message: fmt.Sprintf(m, args...)}
}

// Internal : getStackTrace allows to get the error extended format.
// Each Frame of the error's StackTrace will be printed in detail.
func getStackTrace(err error) string {
	var stack string
	if err, ok := err.(stackTracer); ok {
		stack = fmt.Sprintf("%+v", err.StackTrace())
	}
	return stack
}
