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
	"bufio"
	"bytes"
	"fmt"
	"github.com/ystia/yorc/commands"
	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/spf13/viper"

	"github.com/ystia/yorc/helper/ziputil"
	"gopkg.in/yaml.v2"
)

var cmdConsul *exec.Cmd
var yorcServerShutdownChan chan struct{}
var yorcServerOutputFile *os.File

// setupYorcServer starts a Yorc server
func setupYorcServer(workingDirectoryPath string) error {

	if err := installDependencies(workingDirectoryPath); err != nil {
		return err
	}

	// Update PATH, as Yorc Server expects to find terraform in its PATH

	workDirAbsolutePath, err := filepath.Abs(workingDirectoryPath)
	if err != nil {
		return err
	}
	newPath := fmt.Sprintf("%s:%s", workDirAbsolutePath, os.Getenv("PATH"))
	if err := os.Setenv("PATH", newPath); err != nil {
		return err
	}

	// Starting Consul begore the Yorc server
	cmdConsul, err = startConsul(workingDirectoryPath)
	if err != nil {
		return err
	}

	// Starting Yorc server

	// First creating a Yorc config file defining the infrastructure
	inputInfra := make(map[string]config.DynamicMap)
	inputInfra[strings.ToLower(infrastructureType)] = inputValues.Infrastructure
	serverConfig := config.Configuration{
		WorkingDirectory: workingDirectoryPath,
		Infrastructures:  inputInfra,
	}

	bSlice, err := yaml.Marshal(serverConfig)
	if err != nil {
		return err
	}

	cfgFileName := filepath.Join(workingDirectoryPath, "config.yorc.yaml")
	err = ioutil.WriteFile(cfgFileName, bSlice, 0700)
	if err != nil {
		return err
	}

	viper.SetConfigFile(cfgFileName)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Yorc Server outputs rediected to a file
	outputFileName := filepath.Join(workingDirectoryPath, "yorc.log")
	yorcServerOutputFile, err := os.Create(outputFileName)
	if err != nil {
		return err
	}

	log.SetOutput(yorcServerOutputFile)

	yorcServerShutdownChan = make(chan struct{})
	go func() {
		err = commands.RunServer(yorcServerShutdownChan)
		if err != nil {
			fmt.Println("Error starting Yorc server", err)
		}
		yorcServerOutputFile.Close()
	}()

	err = waitForYorcServerUP(5 * time.Second)

	return err

}

func waitForYorcServerUP(timeout time.Duration) error {

	nbAttempts := timeout / time.Second

	client, err := httputil.GetClient(clientConfig)
	if err != nil {
		return err
	}

	request, err := client.NewRequest("GET", "/deployments", nil)
	if err != nil {
		return err
	}
	request.Header.Add("Accept", "application/json")

	for {
		response, err := client.Do(request)
		if err == nil {
			defer response.Body.Close()
			return nil
		}

		nbAttempts--
		if nbAttempts < 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Timeout waiting %s seconds for Yorc Server to be up", nbAttempts)
}

// tearDownYorcServer tears down a Yorc server
func tearDownYorcServer(workingDirectoryPath string) error {

	// Stop Yorc server
	if yorcServerShutdownChan != nil {
		close(yorcServerShutdownChan)
		yorcServerOutputFile.Close()
	}

	// stop Consul
	if cmdConsul != nil {
		if err := cmdConsul.Process.Kill(); err != nil {
			return err
		}
	}

	return nil

}

// startConsul starts a consul server, returns the command started
func startConsul(workingDirectoryPath string) (*exec.Cmd, error) {

	executable := filepath.Join(workingDirectoryPath, "consul")
	dataDir := filepath.Join(workingDirectoryPath, "consul-data")
	// Consul startup options
	// Specifying a bind address or it would fail on a setup with multiple
	// IPv4 addresses configured
	cmdArgs := "agent -server -bootstrap-expect 1 -bind 127.0.0.1 -data-dir " + dataDir
	cmd := exec.Command(executable, strings.Split(cmdArgs, " ")...)
	output, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	// Wait for a new leader to be elected, else Yorc could try to access Consul
	// when it is not yet ready
	err = waitForOutput(output, "New leader elected", 60*time.Second)
	if err != nil {
		tearDownYorcServer(workingDirectoryPath)
		return nil, err
	}
	return cmd, err
}

func waitForOutput(output io.ReadCloser, expected string, timeout time.Duration) error {

	reader := bufio.NewReader(output)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("Timeout waiting for %s", expected)
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			if strings.Contains(line, expected) {
				return nil
			}
		}
	}

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

		// Check semantic versions if possible
		installedSemanticVersion, errInstalled := semver.Make(installedVersion)
		neededVersion, errNeeded := semver.Make(version)

		if errInstalled != nil || errNeeded != nil {
			if installedVersion == version {
				fmt.Printf("Ansible module %s already installed\n",
					installedVersion)
				return nil
			}
		} else if installedSemanticVersion.GE(neededVersion) {
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
