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
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ZipPath zips a file or directory and return the zipped content as a byte array.
//
// path could be either relative or absolute.
func ZipPath(path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return []byte{}, err
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return []byte{}, err
	}
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return []byte{}, err
	}

	if fileInfo.IsDir() {
		err = zipDirContent(w, "", absPath)
		if err != nil {
			return []byte{}, err
		}
	} else {
		fileName := fileInfo.Name()
		var header *zip.FileHeader
		header, err = zip.FileInfoHeader(fileInfo)
		if err != nil {
			return []byte{}, err
		}
		header.Name = fileName
		// Get a writer in the archive based on our header
		var writer io.Writer
		writer, err = w.CreateHeader(header)
		if err != nil {
			return []byte{}, err
		}
		var file *os.File
		file, err = os.Open(absPath)
		if err != nil {
			return []byte{}, err
		}
		if _, err = io.Copy(writer, file); err != nil {
			return []byte{}, err
		}
	}

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// zipDirContent zips files and directories recursively.
//
// If not empty, rootEntry must end with a forward slash '/'
func zipDirContent(w *zip.Writer, rootEntry, dirPath string) error {
	//tb.Logf("Analyzing %q rootEntry %q", dirPath, rootEntry)
	fileInfos, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		absPath, err := filepath.Abs(filepath.Join(dirPath, fileInfo.Name()))
		if err != nil {
			return err
		}
		absPath, err = filepath.EvalSymlinks(absPath)
		if err != nil {
			return err
		}
		fileName := rootEntry + fileInfo.Name()

		fileInfo, err = os.Stat(absPath)
		if err != nil {
			return err
		}

		// Create a header based off of the fileinfo
		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}

		// If it's a file, set the compression method to deflate (leave directories uncompressed)
		if !fileInfo.IsDir() {
			header.Method = zip.Deflate
		}

		header.Name = fileName

		// Add a trailing slash if the entry is a directory
		if fileInfo.IsDir() {
			header.Name += "/"
		}

		// Get a writer in the archive based on our header
		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			file, err := os.Open(absPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, file); err != nil {
				return err
			}
		} else {
			if err := zipDirContent(w, fileName+"/", absPath); err != nil {
				return err
			}
		}

	}
	return nil
}
