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

package ziputil

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// Unzip unzips a file in a given directory,
// returns the list of paths to files in the destination directory
func Unzip(zipFilePath, destinationPath string) ([]string, error) {

	var destFilenames []string

	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return destFilenames, err
	}
	defer reader.Close()

	for _, f := range reader.File {

		rc, err := f.Open()
		if err != nil {
			return destFilenames, err
		}

		destFilepath := filepath.Join(destinationPath, f.Name)

		destFilenames = append(destFilenames, destFilepath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(destFilepath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(destFilepath), os.ModePerm); err != nil {
				return destFilenames, err
			}

			outFile, err := os.OpenFile(destFilepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return destFilenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()
			rc.Close()

			if err != nil {
				return destFilenames, err
			}

		}
	}
	return destFilenames, nil
}
