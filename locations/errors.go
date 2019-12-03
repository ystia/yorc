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

package locations

import (
	"github.com/pkg/errors"
)

type locationAlreadyExistError struct{}

func (e locationAlreadyExistError) Error() string {
	return "a location with the same name already exists"
}

// IsLocationAlreadyExistError checks if an error is an "location already exists" error
func IsLocationAlreadyExistError(err error) bool {
	_, ok := errors.Cause(err).(locationAlreadyExistError)
	return ok
}

type locationNotFoundError struct{}

func (e locationNotFoundError) Error() string {
	return "location not found"
}

// IsLocationNotFoundError checks if an error is an "location not found" error
func IsLocationNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(locationNotFoundError)
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
