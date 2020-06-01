// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

type hostConnectionError struct {
	message string
}

func (e hostConnectionError) Error() string {
	return e.message
}

// IsHostConnectionError checks if an error is an error due to host connection error
func IsHostConnectionError(err error) bool {
	_, ok := errors.Cause(err).(hostConnectionError)
	return ok
}
