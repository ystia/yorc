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

package bootstrap

import (
	"fmt"

	"github.com/ystia/yorc/helper/ziputil"
	resources "gopkg.in/cookieo9/resources-go.v2"
)

// extractResources extracts resources from the file in argument if any,
// else extract resources embedded in this binary.
// Resources are extracted in resourcesDir directory
func extractResources(resourcesZipFilePath, resourcesDir string) error {

	if resourcesZipFilePath != "" {
		if _, err := ziputil.Unzip(resourcesZipFilePath, resourcesDir); err != nil {
			return err
		}
	} else {
		// Use the embedded resources
		exePath, err := resources.ExecutablePath()
		if err != nil {
			return err
		}
		bundle, err := resources.OpenZip(exePath)
		if err != nil {
			return err
		}
		defer bundle.Close()
		var bundles resources.BundleSequence
		bundles = append(bundles, bundle)
		allResources, err := bundles.List()
		if err != nil {
			return err
		}
		for _, res := range allResources {
			fmt.Printf("Found resource %s\n", res.Path())
		}

		return nil
	}

	return nil
}
