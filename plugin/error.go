package plugin

import (
	"fmt"
	"github.com/pkg/errors"
)

// PluginError is a wrapper dedicated to allow passing errors via RPC encoding
// It must be used for RPC server side in the response call method (ie plugin side actually)
type PluginError struct {
	Message string
	Stack   string
}

// This allows PluginError to implement error
func (pErr *PluginError) Error() string {
	return fmt.Sprintf("Plugin error with message:%s and stack:%s", pErr.Message, pErr.Stack)
}

// This allows to cast PluginError to error keeping all error details
func toError(pErr *PluginError) error {
	if pErr != nil {
		return errors.Wrap(errors.New(pErr.Message), pErr.Stack)
	}
	return nil
}

// This allows to instantiate a PluginError from an error builtin type
func NewPluginError(err error) *PluginError {
	return &PluginError{Message: err.Error(), Stack: getStackTrace(err)}
}

// This allows to instantiate a PluginError from a message and variadic arguments
func NewPluginErrorFromMessage(m string, args ...interface{}) *PluginError {
	return &PluginError{Message: fmt.Sprintf(m, args...)}
}

// This allows to instantiate a PluginError from an initial error and an additional message
func NewPluginErrorFromMessageAndCause(err error, m string) *PluginError {
	return &PluginError{Message: m, Stack: getStackTrace(err)}
}

// Internal : allows to get the error extended format.
// Each Frame of the error's StackTrace will be printed in detail.
func getStackTrace(err error) string {
	var stack string
	if err != nil {
		stack = fmt.Sprintf("%+v", err)
	}
	return stack
}
