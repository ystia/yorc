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

package pathutil

import (
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"os"
)

// IsValidPath checks if a string is a valid path. (~ is handled)
func IsValidPath(str string) (bool, error) {
	expPath, err := homedir.Expand(str)
	if err != nil {
		return false, errors.Wrapf(err, "failed to expand path:%q", str)
	}

	if _, err := os.Stat(expPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}
