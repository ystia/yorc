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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/blang/semver"

	"github.com/ystia/yorc/helper/ziputil"
)

var consulPID int

// setupYorcServer starts a Yorc server
func setupYorcServer(workingDirectoryPath string) error {

	if err := installDependencies(workingDirectoryPath); err != nil {
		return err
	}

	var err error
	consulPID, err = startConsul(workingDirectoryPath)
	if err != nil {
		return err
	}
	return nil

	
}

// startConsul starts a consul server, returns the process ID
func startConsul(workingDirectoryPath string) (int, error) {

	executable := filepath.Join(workingDirectoryPath, "consul")
	dataDir := filepath.Join(workingDirectoryPath, "consul-data")
	cmdArgs := "agent -server -bootstrap-expect 1 -data-dir " + dataDir
	cmd := exec.Command(executable, strings.Split(cmdArgs, " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return -1, err
	}

	return cmd.Process.Pid, err
}

// downloadDependencies downloads Yorc Server dependencies
func installDependencies(workingDirectoryPath string) error {

	// Install Ansible
	ansibleVersion := inputValues.Ansible.Version
	if err := installAnsible(ansibleVersion); err != nil {
		return err
	}

	// Download Consul
	url := inputValues.Consul.DownloadURL
	if err := downloadUnzip(url, workingDirectoryPath); err != nil {
		return err
	}

	// Download Terraform
	version := inputValues.Terraform.Version
	url = fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_amd64.zip", version, version)
	if err := downloadUnzip(url, workingDirectoryPath); err != nil {
		return err
	}

	// Donwload Terraform plugins

	return nil
}

// getFilePath returns the destination file path from a URL to download and a destination directory
func getFilePath(url, destinationPath string) string {
	_, file := filepath.Split(url)
	return filepath.Join(destinationPath, file)
}

// download donwloads and unzips a URL to a given destination
func downloadUnzip(url, destinationPath string) error {

	filePath := getFilePath(url, destinationPath)

	fmt.Println("Downloading", url, "to", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// The file was not downloaded yet
		outputFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		response, err := http.Get(url)
		if err != nil {
			outputFile.Close()
			return err
		}
		defer response.Body.Close()

		_, err = io.Copy(outputFile, response.Body)
		outputFile.Close()
		if err != nil {
			return err
		}
	}

	// File downloaded, unzipping it
	_, err := ziputil.Unzip(filePath, destinationPath)
	return err
}

// installAnsible installs the Ansible version in argument, using pip3 if available
// else using pip
func installAnsible(version string) error {
	pipCmd, err := getPipCmd()
	if err != nil {
		return err
	}

	// Check if ansible is already installed
	installedVersion, err := getPipModuleInstalledVersion(pipCmd, "ansible")
	if err == nil && installedVersion != "" {

		installedSemanticVersion, err := semver.Make(installedVersion)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse ansible installed version %s", installedVersion)
		}
		neededVersion, err := semver.Make(version)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse ansible needed version %s", neededVersion)
		}

		if installedSemanticVersion.GE(neededVersion) {
			fmt.Printf("Not installing ansible module as installed version %s >= needed version %s\n",
				installedVersion, version)
			return nil
		}

	}

	// Installing ansible
	cmd := exec.Command(pipCmd, "install", "--upgrade", "ansible=="+version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

// getPipModuleInstalledVersion retruns the version of an installed pip module.
// Returns an error if the module is not installed
func getPipModuleInstalledVersion(pipCmd, module string) (string, error) {
	cmd := exec.Command(pipCmd, "show", module)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}
	output := string(cmdOutput.Bytes())
	match := regexp.MustCompile("Version: (\\w.+)").FindStringSubmatch(output)
	version := ""
	if match != nil {
		version = match[1]
	}

	return version, nil
}

// getPipCmd checks if pip3 is available and returns this command, else checks
// if pip is avaible and returns this command, else returns an error
func getPipCmd() (string, error) {
	pipCmds := []string{"pip3", "pip"}

	for _, cmdName := range pipCmds {
		cmd := exec.Command(cmdName, "--version")
		if err := cmd.Run(); err == nil {
			return cmdName, nil
		}
	}

	return "", fmt.Errorf("ERROR: didn't find pip3 or pip installed")
}
