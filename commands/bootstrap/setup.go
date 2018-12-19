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
	"encoding/json"
	"fmt"
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

	"github.com/ystia/yorc/commands/httputil"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/ziputil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/rest"

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

	// First creating a Yorc config file defining the infrastructure
	serverConfig := config.Configuration{
		WorkingDirectory: workingDirectoryPath,
		ResourcesPrefix:  "bootstrap-",
		WorkersNumber:    inputValues.Yorc.WorkersNumber,
		Infrastructures:  inputValues.Infrastructures,
		Terraform: config.Terraform{
			PluginsDir: workDirAbsolutePath,
		},
		Ansible: config.Ansible{
			DebugExec:            true,
			KeepGeneratedRecipes: true,
		},
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

	// Yorc Server outputs redirected to a file
	outputFileName := filepath.Join(workingDirectoryPath, "yorc.log")
	yorcServerOutputFile, err := os.OpenFile(outputFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	log.SetOutput(yorcServerOutputFile)

	// Get current executable path
	yorcExecutablePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Before starting yorc, hceck if it is already running
	// Starting Yorc server
	err = waitForYorcServerUP(0)
	if err == nil {
		fmt.Println("Local Yorc server already started")
	} else {

		fmt.Println("Starting a local Yorc server to bootstrap a remote Yorc server...")

		cmdArgs := fmt.Sprintf("server --config %s --http_port %d",
			cfgFileName, inputValues.Yorc.Port)
		cmd := exec.Command(yorcExecutablePath, strings.Split(cmdArgs, " ")...)
		cmd.Stdout = yorcServerOutputFile
		err = cmd.Start()
		if err != nil {
			return err
		}

		err = waitForYorcServerUP(30 * time.Second)
		if err != nil {
			return err
		}
		fmt.Println("...local Yorc server started")
	}

	// For a deployment on a Hosts Pool, this Hosts Pool needs to be configured,
	// which can be done now that the local Yorc Server is up
	if infrastructureType == "hostspool" {

		fmt.Println("Configuring a Hosts Pool")
		client, err := getYorcClient()
		if err != nil {
			return err
		}

		// Use the local path to Yorc key to connect to hosts

		for i := range inputValues.Hosts {
			inputValues.Hosts[i].Connection.PrivateKey = inputValues.Yorc.PrivateKeyFile
		}

		hostsPool := rest.HostsPoolRequest{
			Hosts: inputValues.Hosts,
		}

		bArray, err := json.Marshal(hostsPool)
		if err != nil {
			return err
		}

		request, err := client.NewRequest("PUT", "/hosts_pool",
			bytes.NewBuffer(bArray))
		if err != nil {
			return err
		}
		request.Header.Add("Content-Type", "application/json")

		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			return err
		}

		httputil.HandleHTTPStatusCode(
			response, "apply", "host pool", http.StatusOK, http.StatusCreated)

	}

	return err

}

func getYorcClient() (*httputil.YorcClient, error) {
	clientConfig := config.Client{YorcAPI: fmt.Sprintf("localhost:%d", inputValues.Yorc.Port)}
	return httputil.GetClient(clientConfig)
}

func waitForYorcServerUP(timeout time.Duration) error {

	nbAttempts := timeout / time.Second

	client, err := getYorcClient()
	if err != nil {
		return err
	}

	request, err := client.NewRequest("PUT", "/hosts_pool", nil)
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

	return fmt.Errorf("Timeout waiting %d seconds for Yorc Server to be up", timeout/time.Second)
}

// cleanBootstrapSetup stops yorc and consul processes, cleans working directories
func cleanBootstrapSetup(workingDirectoryPath string) error {

	// Stop Yorc server
	if yorcServerShutdownChan != nil {
		close(yorcServerShutdownChan)
		yorcServerOutputFile.Close()
	} else {
		cmd := exec.Command("pkill", "-f", "yorc server")
		cmd.Run()
	}

	// stop Consul
	if cmdConsul != nil {
		if err := cmdConsul.Process.Kill(); err != nil {
			return err
		}
	} else {
		cmd := exec.Command("pkill", "consul")
		cmd.Run()

	}

	// Clean working directories
	os.RemoveAll(filepath.Join(workingDirectoryPath, "bootstrapResources"))
	os.RemoveAll(filepath.Join(workingDirectoryPath, "deployments"))
	os.RemoveAll(filepath.Join(workingDirectoryPath, "consul-data"))
	os.Remove(filepath.Join(workingDirectoryPath, "config.yorc.yaml"))
	os.Remove(filepath.Join(workingDirectoryPath, "yorc.log"))

	fmt.Println("Local setup cleaned up")
	return nil

}

// startConsul starts a consul server, returns the command started
func startConsul(workingDirectoryPath string) (*exec.Cmd, error) {

	executable := filepath.Join(workingDirectoryPath, "consul")
	dataDir := filepath.Join(workingDirectoryPath, "consul-data")

	// First check if consul is already running
	cmd := exec.Command(executable, "members")
	err := cmd.Run()
	if err == nil {
		fmt.Println("Consul is already running")
		return nil, nil
	}

	fmt.Println("Starting Consul...")
	// Consul startup options
	// Specifying a bind address or it would fail on a setup with multiple
	// IPv4 addresses configured
	cmdArgs := "agent -server -bootstrap-expect 1 -bind 127.0.0.1 -data-dir " + dataDir
	cmd = exec.Command(executable, strings.Split(cmdArgs, " ")...)
	output, _ := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	// Wait for a new leader to be elected, else Yorc could try to access Consul
	// when it is not yet ready
	err = waitForOutput(output, "New leader elected", 60*time.Second)
	if err != nil {
		cleanBootstrapSetup(workingDirectoryPath)
		return nil, err
	}

	fmt.Println("...Consul started")
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
	if err := downloadUnzip(inputValues.Consul.DownloadURL, "consul.zip", workingDirectoryPath); err != nil {
		return err
	}

	// Download Terraform
	if err := downloadUnzip(inputValues.Terraform.DownloadURL, "terraform.zip", workingDirectoryPath); err != nil {
		return err
	}

	// Donwload Terraform plugins
	for i, url := range inputValues.Terraform.PluginURLs {
		if err := downloadUnzip(url, fmt.Sprintf("terraform-plugin-%d.zip", i), workingDirectoryPath); err != nil {
			return err
		}
	}

	return nil
}

// getFilePath returns the destination file path from a URL to download and a destination directory
func getFilePath(url, destinationPath string) string {
	_, file := filepath.Split(url)
	return filepath.Join(destinationPath, file)
}

// downloadUnzip donwloads and unzips a URL to a given destination
func downloadUnzip(url, fileName, destinationPath string) error {
	filePath, err := download(url, fileName, destinationPath)
	if err != nil {
		return err
	}
	// File downloaded, unzipping it
	_, err = ziputil.Unzip(filePath, destinationPath)
	if err != nil && strings.Contains(err.Error(), "text file busy") {
		// Unzip of binary already in use, ignoring
		return nil
	}
	return err
}

// download donwloads and optionally unzips a URL to a given destination
// returns the path to destination file
func download(url, fileName, destinationPath string) (string, error) {

	filePath := filepath.Join(destinationPath, fileName)

	fmt.Println("Downloading", url, "to", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// The file was not downloaded yet
		outputFile, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer outputFile.Close()

		response, err := http.Get(url)
		if err != nil {
			outputFile.Close()
			return filePath, err
		}
		defer response.Body.Close()

		_, err = io.Copy(outputFile, response.Body)
		outputFile.Close()
		if err != nil {
			return filePath, err
		}
	}

	return filePath, nil
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
