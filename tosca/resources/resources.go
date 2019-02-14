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

package resources

import (
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	resources "gopkg.in/cookieo9/resources-go.v2"
)

func getToscaResources() (map[string][]byte, error) {
	// try to get resources from Yorc executable
	// Use the embedded resources
	exePath, err := resources.ExecutablePath()
	if err != nil {
		return nil, err
	}
	bundle, err := resources.OpenZip(exePath)
	if err != nil {
		return nil, errors.Wrap(err, "can't find TOSCA resources in yorc executable, not a valid zip file")
	}
	defer bundle.Close()
	searcher := bundle.(resources.Searcher)
	toscaResources, err := searcher.Glob("tosca/*.yml")
	if err != nil {
		return nil, errors.Wrap(err, "can't find TOSCA resources in yorc executable")
	}
	res := make(map[string][]byte, len(toscaResources))
	for _, r := range toscaResources {
		// Use path instead of filepath as it is platform independent
		rName := path.Base(r.Path())
		reader, err := r.Open()
		if err != nil {
			return nil, errors.Wrapf(err, "can't open local TOSCA resource %s", rName)
		}
		rContent, err := ioutil.ReadAll(reader)
		reader.Close()
		if err != nil {
			return nil, errors.Wrapf(err, "can't read local TOSCA resource %s", rName)
		}
		res[rName] = rContent
	}
	return res, nil
}
