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
	"encoding/json"
	"errors"
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
	"gopkg.in/yaml.v2"

	"github.com/cheggaaa/pb/v3"
	"github.com/ystia/yorc/v4/commands/httputil"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/ziputil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/rest"
)

var cmdConsul *exec.Cmd
var yorcServerShutdownChan chan struct{}
var yorcServerOutputFile *os.File

type dependenciesVersion struct {
	Alien4cloud string
	Consul      string
	Terraform   string
	Yorc        string
}

var previousBootstrapVersion dependenciesVersion

// setupYorcServer starts a Yorc server
func setupYorcServer(workingDirectoryPath string) error {

	// Get previous version of Yorc used for the bootstrap
	// to known if dependencies
	previousBootstrapVersion = getPreviousSetupDependenciesVersion(workingDirectoryPath)
	if err := installDependencies(workingDirectoryPath); err != nil {
		return err
	}

	// Store new versions installed
	storeDependenciesVersion(workingDirectoryPath)

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

	// First creating a config file defining the locations
	locationsFilePath := ""
	if len(inputValues.Locations) > 0 {
		locs := locations.LocationsDefinition{Locations: inputValues.Locations}
		bSlice, err := yaml.Marshal(locs)
		if err != nil {
			return err
		}
		locationsFilePath = filepath.Join(workingDirectoryPath, "locations.yorc.yaml")
		err = ioutil.WriteFile(locationsFilePath, bSlice, 0700)
		if err != nil {
			return err
		}
	}

	// Creating a Yorc config file defining the infrastructure
	serverConfig := config.Configuration{
		WorkingDirectory:  workingDirectoryPath,
		ResourcesPrefix:   "bootstrap-",
		WorkersNumber:     inputValues.Yorc.WorkersNumber,
		LocationsFilePath: locationsFilePath,
		Terraform: config.Terraform{
			PluginsDir:         workDirAbsolutePath,
			KeepGeneratedFiles: true,
		},
		Ansible: config.Ansible{
			DebugExec:            true,
			KeepGeneratedRecipes: true,
			UseOpenSSH:           inputValues.Ansible.UseOpenSSH,
			Inventory:            inputValues.Ansible.Inventory,
			HostedOperations:     config.HostedOperations{UnsandboxedOperationsAllowed: inputValues.Ansible.HostOperationsAllowed},
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

	// Before starting yorc, check if it is already running
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

		request, err := client.NewRequest("PUT", "/hosts_pool/"+locationName,
			bytes.NewBuffer(bArray))
		if err != nil {
			return err
		}
		request.Header.Add("Content-Type", "application/json")

		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		httputil.HandleHTTPStatusCode(
			response, "apply", "host pool", http.StatusOK, http.StatusCreated)

	}

	return err

}

// getPreviousSetupDependenciesVersion returns versions of components
// donwloaded for a previous bootstrap
func getPreviousSetupDependenciesVersion(workingDirectoryPath string) dependenciesVersion {
	var versions dependenciesVersion
	versionsFile := filepath.Join(workingDirectoryPath, "versions.yaml")
	if _, err := os.Stat(versionsFile); os.IsNotExist(err) {
		// Previous version didn't store its versions
		// will return empty values, so that the current bootstrap will
		// download dependencies
		return versions
	}

	yamlFile, err := ioutil.ReadFile(versionsFile)
	if err != nil {
		fmt.Printf("Overriding previous dependencies on error reading file %s: %v\n", yamlFile, err)
		return versions
	}
	err = yaml.Unmarshal(yamlFile, &versions)
	if err != nil {
		fmt.Printf("Overriding previous dependencies on error unmarshaling file %s: %v\n", yamlFile, err)
	}

	return versions
}

// storeDependenciesVersion sotres in the working directory versions of components
// donwloaded by the bootstrap
func storeDependenciesVersion(workingDirectoryPath string) {

	versions := dependenciesVersion{
		Alien4cloud: alien4cloudVersion,
		Consul:      consulVersion,
		Terraform:   terraformVersion,
		Yorc:        yorcVersion,
	}

	bSlice, err := yaml.Marshal(versions)
	if err != nil {
		fmt.Printf("Failed to marshall dependencies versions %v\n", err)
		return
	}

	filename := filepath.Join(workingDirectoryPath, "versions.yaml")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		fmt.Printf("Failed to open file file %s: %v\n", filename, err)
		return
	}

	defer file.Close()

	_, err = fmt.Fprintf(file, string(bSlice[:]))

	if err != nil {
		fmt.Printf("Failed to write in file %s: %v\n", filename, err)
	}
}

func getYorcClient() (httputil.HTTPClient, error) {
	clientConfig := config.Client{YorcAPI: fmt.Sprintf("localhost:%d", inputValues.Yorc.Port)}
	return httputil.GetClient(clientConfig)
}

func waitForYorcServerUP(timeout time.Duration) error {

	nbAttempts := timeout / time.Second

	client, err := getYorcClient()
	if err != nil {
		return err
	}

	request, err := client.NewRequest("GET", "/hosts_pool", nil)
	if err != nil {
		return err
	}
	request.Header.Add("Accept", "application/json")

	for {
		response, err := client.Do(request)
		if err == nil {
			response.Body.Close()
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
	os.Remove(filepath.Join(workingDirectoryPath, "locations.yorc.yaml"))
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
	consulLogPath := filepath.Join(workingDirectoryPath, "consul.log")
	consulLogFile, err := os.OpenFile(consulLogPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Starting Consul (logs: %s)", consulLogPath)
	// Consul startup options
	// Specifying a bind address or it would fail on a setup with multiple
	// IPv4 addresses configured
	cmdArgs := "agent -server -bootstrap-expect 1 -bind 127.0.0.1 -data-dir " + dataDir
	cmd = exec.Command(executable, strings.Split(cmdArgs, " ")...)
	cmd.Stdout = consulLogFile
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	// Wait for a new leader to be elected, else Yorc could try to access Consul
	// when it is not yet ready
	waitForConsulReadiness("http://127.0.0.1:8500")
	fmt.Println(" Consul started!")
	return cmd, err
}

func waitForConsulReadiness(consulHTTPEndpoint string) {
	for {
		fmt.Print(".")
		leader, _ := getConsulLeader(consulHTTPEndpoint)
		if leader != "" {
			return
		}
		<-time.After(2 * time.Second)
	}
}

func getConsulLeader(consulHTTPEndpoint string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1/status/leader", consulHTTPEndpoint))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	matches := regexp.MustCompile(`"(.*)"`).FindStringSubmatch(string(bodyBytes))
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], nil
}

// downloadDependencies downloads Yorc Server dependencies
func installDependencies(workingDirectoryPath string) error {

	// Install Ansible
	ansibleVersion := inputValues.Ansible.Version
	if err := installAnsible(ansibleVersion); err != nil {
		return err
	}

	// Install paramiko
	if err := installParamiko(); err != nil {
		return err
	}

	// Download Consul
	overwrite := (previousBootstrapVersion.Consul != consulVersion)
	if err := downloadUnzip(inputValues.Consul.DownloadURL, "consul.zip", workingDirectoryPath, overwrite); err != nil {
		return err
	}

	// Download Terraform
	overwrite = (previousBootstrapVersion.Terraform != terraformVersion)
	if err := downloadUnzip(inputValues.Terraform.DownloadURL, "terraform.zip", workingDirectoryPath, overwrite); err != nil {
		return err
	}

	// Donwload Terraform plugins
	for i, url := range inputValues.Terraform.PluginURLs {
		if err := downloadUnzip(url, fmt.Sprintf("terraform-plugin-%d.zip", i), workingDirectoryPath, overwrite); err != nil {
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
func downloadUnzip(url, fileName, destinationPath string, overwrite bool) error {
	filePath, err := download(url, fileName, destinationPath, overwrite)
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
func download(url, fileName, destinationPath string, overwrite bool) (string, error) {

	filePath := filepath.Join(destinationPath, fileName)

	fmt.Println("Downloading", url, "to", filePath)

	execute := overwrite
	if !overwrite {
		_, err := os.Stat(filePath)
		execute = os.IsNotExist(err)
	}
	if execute {
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

		//bar
		bar := pb.Full.Start(int(response.ContentLength))
		barReader := bar.NewProxyReader(response.Body)

		_, err = io.Copy(outputFile, barReader)
		outputFile.Close()
		bar.Finish()
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
		} else {
			noUpgradeVersion, _ := semver.Make("2.10.0")
			if installedSemanticVersion.LT(noUpgradeVersion) {
				fmt.Printf("No ansible upgrade possible from %s to %s, uninstalling %s first\n",
					installedVersion, version, installedVersion)
				cmd := exec.Command(pipCmd, "uninstall", "ansible")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					return err
				}
			}
		}

	}

	// Installing ansible
	cmd := exec.Command(pipCmd, "install", "--upgrade", "ansible=="+version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

// installParamiko installs Paramiko needed when openSSH is not used
// Install using pip3 if available else using pip
func installParamiko() error {
	pipCmd, err := getPipCmd()
	if err != nil {
		return err
	}

	// Installing ansible
	cmd := exec.Command(pipCmd, "install", "--upgrade", "paramiko")
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
