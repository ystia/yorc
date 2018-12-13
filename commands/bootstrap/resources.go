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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
		for _, resource := range allResources {
			if !strings.HasPrefix(resource.Path(), "bootstrap/resources") {
				continue
			}

			destinationPath := filepath.Join(resourcesDir,
				strings.TrimPrefix(resource.Path(), "bootstrap/resources"))

			info, err := resource.Stat()
			if err != nil {
				return err
			}

			if info.IsDir() {
				continue
			}

			// This is a file to copy
			src, err := resource.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			if err := os.MkdirAll(filepath.Dir(destinationPath), 0700); err != nil {
				return err
			}

			dest, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}
			defer dest.Close()

			if _, err := io.Copy(dest, src); err != nil {
				return err
			}

			// If the copied file is a zip, unzipping it
			if filepath.Ext(destinationPath) == ".zip" {
				if _, err := ziputil.Unzip(destinationPath, filepath.Dir(destinationPath)); err != nil {
					return err
				}
			}

		}

		return nil
	}

	return nil
}

// getAlien4CloudVersionFromTOSCATypes returns the ALien4Cloud version from the
// bundled resources zip file containing TOSCA types needed for the bootstrap
func getAlien4CloudVersionFromTOSCATypes() string {

	return getVersionFromTOSCATypes("alien-base-types")
}

// getForgeDefaultVersion returns the Forge version from the bundled
// resources zip file containing TOSCA types needed for the bootstrap
func getForgeVersionFromTOSCATypes() string {

	return getVersionFromTOSCATypes("org.ystia.yorc.pub")
}

// getVersionFromTOSCATypes returns the version from the bundled
// resources zip file containing TOSCA types needed for the bootstrap
func getVersionFromTOSCATypes(path string) string {
	// Use the embedded resources
	version := "unknown"
	exePath, err := resources.ExecutablePath()
	if err != nil {
		fmt.Println("Failed to get resources bundle", err)
		return version
	}
	bundle, err := resources.OpenZip(exePath)
	if err != nil {
		fmt.Println("Failed to open resources bundle", err)
		return version
	}
	defer bundle.Close()
	var bundles resources.BundleSequence
	bundles = append(bundles, bundle)
	allResources, err := bundles.List()
	if err != nil {
		fmt.Println("Failed to list resources bundle content", err)
		return version
	}

	re := regexp.MustCompile(path + "/" + `([0-9a-zA-Z.-]+)/`)
	for _, resource := range allResources {
		if !strings.HasSuffix(resource.Path(), "tosca_types.zip") {
			continue
		}

		src, err := resource.Open()
		if err != nil {
			fmt.Println("Failed to open tosca types zip in bundled resources", err)
			return version
		}
		defer src.Close()

		allRead, err := ioutil.ReadAll(src)
		if err != nil {
			fmt.Println("Failed to read tosca types zip in bundled resources", err)
			return version
		}

		match := re.FindStringSubmatch(string(allRead[:]))
		if match != nil {
			version = match[1]
			return version
		}
	}

	return version

}
