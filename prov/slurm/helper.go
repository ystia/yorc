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

package slurm

import (
	"bufio"
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/sshutil"
	"github.com/ystia/yorc/v3/log"
)

const reSbatch = `^Submitted batch job (\d+)`

const invalidJob = "Invalid job id specified"

// getSSHClient returns a SSH client with slurm credentials from node or job configuration provided by the deployment,
// or by the yorc slurm configuration
func getSSHClient(userName string, privateKey string, password string, cfg config.Configuration) (*sshutil.SSHClient, error) {
	// Check manadatory slurm configuration
	if err := checkInfraConfig(cfg); err != nil {
		log.Printf("Unable to provide SSH client due to:%+v", err)
		return nil, err
	}

	// Get user credentials provided by the deployment, if any
	if userName != "" {
		if password == "" && privateKey == "" {
			return nil, errors.New("Slurm missing authentication details in deployment properties, password or private_key should be set")
		}
	} else {
		// Get user credentials from the yorc configuration
		if err := checkInfraUserConfig(cfg); err != nil {
			log.Printf("Unable to provide SSH client due to:%+v", err)
			return nil, err
		}
		userName = strings.Trim(cfg.Infrastructures[infrastructureName].GetString("user_name"), "")
		privateKey = strings.Trim(cfg.Infrastructures[infrastructureName].GetString("private_key"), "")
		password = strings.Trim(cfg.Infrastructures[infrastructureName].GetString("password"), "")
	}

	// Get SSH client
	SSHConfig := &ssh.ClientConfig{
		User:            userName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Set an authentication method. At least one authentication method
	// has to be set, private/public key or password.
	if privateKey != "" {
		keyAuth, err := sshutil.ReadPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		SSHConfig.Auth = append(SSHConfig.Auth, keyAuth)
	}

	if password != "" {
		SSHConfig.Auth = append(SSHConfig.Auth, ssh.Password(password))
	}

	port, err := strconv.Atoi(cfg.Infrastructures[infrastructureName].GetString("port"))
	if err != nil {
		wrapErr := errors.Wrap(err, "slurm configuration port is not a valid port")
		log.Printf("Unable to provide SSH client due to:%+v", wrapErr)
		return nil, err
	}

	return &sshutil.SSHClient{
		Config: SSHConfig,
		Host:   cfg.Infrastructures[infrastructureName].GetString("url"),
		Port:   port,
	}, nil
}

// getUserCredentials returns user credentials from a node property, or a capability property.
// the property name is provided by propertyName parameter, and its type is supposed to be tosca.datatypes.Credential
func getUserCredentials(kv *api.KV, deploymentID string, nodeName string, capabilityName, propertyName string) (*UserCredentials, error) {
	// Check if user credentials provided in node definition
	userName, err := getPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, "user")
	if err != nil {
		return nil, err
	}
	if userName != "" {
		log.Debugf("Got user name %s from property %s", userName, propertyName)
	}

	// Check for token-type
	tokenType, err := getPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, "token_type")
	if err != nil {
		return nil, err
	}

	log.Debugf("Got token_type %s from property %s", tokenType, propertyName)

	var password, privateKey string
	switch tokenType {
	case "password":
		password, err = getPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, "token")
		if err != nil {
			return nil, err
		}
		if password != "" {
			log.Debugf("Got password from property")
		}
	case "private_key":
		privateKey, err = getPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, "keys", "0")
		if err != nil {
			return nil, err
		}
		if privateKey != "" {
			log.Debugf("Got private key from property")
		}
	default:
		// password or private_key expected as token_type
		if capabilityName != "" {
			return nil, errors.Errorf("Unsupported token_type %s in capability %s property %s. One of password or private_key expected", tokenType, capabilityName, propertyName)
		} else {
			return nil, errors.Errorf("Unsupported token_type %s in property %s. One of password or private_key expected", tokenType, propertyName)
		}

	}

	return &UserCredentials{UserName: userName, PrivateKey: privateKey, Password: password}, nil

}

func getPropertyValue(kv *api.KV, deploymentID string, nodeName string, capabilityName string, propertyName string, nestedKeys ...string) (string, error) {
	var value string
	var propValue *deployments.TOSCAValue
	var err error
	if capabilityName != "" {
		if len(nestedKeys) == 1 {
			propValue, err = deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, nestedKeys[0])
		} else {
			propValue, err = deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, nestedKeys[0], nestedKeys[1])
		}
	} else {
		if len(nestedKeys) == 1 {
			propValue, err = deployments.GetNodePropertyValue(kv, deploymentID, nodeName, propertyName, nestedKeys[0])
		} else {
			propValue, err = deployments.GetNodePropertyValue(kv, deploymentID, nodeName, propertyName, nestedKeys[0], nestedKeys[1])
		}
	}
	if err != nil {
		return "", err
	}
	if propValue != nil {
		value = propValue.RawString()
	}
	return value, nil
}

// checkInfraConfig checks slurm infrastructure mandatory configuration parameters :
// - url (slurm client's node address)
// - port (slurm client's node port)
// returns error in case of inconsistent configuration, or nil if configuration ok
func checkInfraConfig(cfg config.Configuration) error {
	_, exist := cfg.Infrastructures[infrastructureName]
	if !exist {
		return errors.New("no slurm infrastructure configuration found")
	}

	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("url"), "") == "" {
		return errors.New("slurm infrastructure url is not set")
	}

	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("port"), "") == "" {
		return errors.New("slurm infrastructure port is not set")
	}

	return nil
}

// checkInfraUserConfig checks slurm infrastructure configuration parameters related to user credentials
// necessary for connect using ssh to the slurm's client node
// - user_name
// - password or private_key
// returns error in case of inconsistent configuration, or nil if configuration seems ok
func checkInfraUserConfig(cfg config.Configuration) error {
	_, exist := cfg.Infrastructures[infrastructureName]
	if !exist {
		return errors.New("no slurm infrastructure configuration found")
	}

	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("user_name"), "") == "" {
		return errors.New("slurm infrastructure user_name is not set")
	}

	// Check an authentication method was specified
	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("password"), "") == "" &&
		strings.Trim(cfg.Infrastructures[infrastructureName].GetString("private_key"), "") == "" {
		return errors.New("slurm infrastructure missing authentication details, password or private_key should be set")
	}

	return nil
}

// getAttributes allows to return an attribute with defined key from specific treatment
func getAttributes(client sshutil.Client, key string, params ...string) ([]string, error) {
	var ret []string
	switch key {
	case "cuda_visible_devices":
		if len(params) == 2 && params[0] != "" {
			cmd := fmt.Sprintf("srun --jobid=%s  bash -c 'env|grep CUDA_VISIBLE_DEVICES'", params[0])
			stdout, err := client.RunCommand(cmd)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to retrieve (%s) for node:%q", key, params[1])
			}
			stdout = strings.Trim(stdout, "\r\n")
			value, err := getEnvValue(stdout)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to retrieve (%s) for node:%q", key, params[1])
			}
			return []string{value}, nil
		}
	case "node_partition":
		if len(params) == 1 && params[0] != "" {
			cmd := fmt.Sprintf("squeue -j %s --noheader -o \"%%N,%%P\"", params[0])
			out, err := client.RunCommand(cmd)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to retrieve Slurm node name/partition")
			}
			split := strings.Split(out, ",")
			if len(split) != 2 {
				return nil, errors.Errorf("Slurm returned an unexpected stdout: %q with command:%q", out, cmd)
			}
			node := strings.Trim(split[0], "\" \t\n\x00")
			part := strings.Trim(split[1], "\" \t\n\x00")
			return []string{node, part}, nil
		}
	default:
		return ret, errors.Errorf("unknown key:%s", key)
	}
	return nil, errors.Errorf("Number of parameters (%d) not as expected for key:%q", len(params), key)
}

// getEnvValue allows to return the value in a formatted string as "property=value"
func getEnvValue(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if strings.ContainsRune(s, '=') {
		propVal := strings.Split(s, "=")
		if len(propVal) == 2 {
			return propVal[1], nil
		}
		return "", errors.New("property/value is malformed")
	}
	return "", errors.New("property/value is malformed")
}

// parseSallocResponse parses stderr and stdout for salloc command
// Below are classic examples:
// salloc: Granted job allocation 1881
// salloc: Pending job allocation 1881

// salloc: Job allocation 1882 has been revoked.
// salloc: error: CPU count per node can not be satisfied
// salloc: error: Job submit/allocate failed: Requested node configuration is not available
func parseSallocResponse(r io.Reader, chRes chan allocationResponse, chErr chan error) {
	var (
		jobID  string
		err    error
		strErr string
	)
	reGranted := regexp.MustCompile(reSallocGranted)
	rePending := regexp.MustCompile(reSallocPending)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if reGranted.MatchString(line) {
			// expected line: "salloc: Granted job allocation 1881"
			if jobID, err = parseJobID(line, reGranted); err != nil {
				chErr <- err
				return
			}
			chRes <- allocationResponse{jobID: jobID, granted: true}
			return
		} else if rePending.MatchString(line) {
			// expected line: "salloc: Pending job allocation 1881"
			if jobID, err = parseJobID(line, rePending); err != nil {
				chErr <- err
				return
			}
			chRes <- allocationResponse{jobID: jobID, granted: false}
			return
		}
		// If no expected lines found, we retrieve the full lines
		if strErr != "" {
			strErr += " "
		}
		strErr += line
	}
	if err := scanner.Err(); err != nil {
		chErr <- errors.Wrap(err, "An error occurred scanning stdout/stderr")
		return
	}
	if len(strErr) > 0 {
		chErr <- errors.Errorf("salloc command returned an error:%q", strErr)
	}
	return
}

func parseJobInfo(r io.Reader) (map[string]string, error) {
	data := make(map[string]string, 0)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		props := strings.Split(line, " ")
		for _, prop := range props {
			if strings.Contains(prop, "=") {
				t := strings.Split(prop, "=")
				data[strings.TrimSpace(t[0])] = strings.TrimSpace(t[1])
			}
		}
	}
	return data, nil
}

func parseJobID(str string, regexp *regexp.Regexp) (string, error) {
	subMatch := regexp.FindStringSubmatch(str)
	if subMatch != nil && len(subMatch) == 2 {
		return subMatch[1], nil
	}
	return "", errors.Errorf("Unable to parse std:%q for retrieving jobID", str)
}

func cancelJobID(jobID string, client *sshutil.SSHClient) error {
	scancelCmd := fmt.Sprintf("scancel %s", jobID)
	sCancelOutput, err := client.RunCommand(scancelCmd)
	if err != nil {
		return errors.Wrapf(err, "Failed to cancel Slurm job: %s:", sCancelOutput)
	}
	return nil
}

func retrieveJobID(out string) (string, error) {
	// expected: "Submitted batch job 4507"
	reBatch := regexp.MustCompile(reSbatch)
	if !reBatch.MatchString(out) {
		return "", errors.Errorf("Unable to parse Job ID from stdout:%q", out)
	}
	jobID, err := parseJobID(out, reBatch)
	if err != nil {
		return "", err
	}
	return jobID, nil
}

func parseKeyValue(str string) (bool, string, string) {
	keyVal := strings.Split(str, "=")
	if len(keyVal) == 2 && strings.TrimSpace(keyVal[0]) != "" && strings.TrimSpace(keyVal[1]) != "" {
		return true, keyVal[0], keyVal[1]
	}
	return false, "", ""
}

func getJobInfo(client sshutil.Client, jobID string) (map[string]string, error) {
	cmd := fmt.Sprintf("scontrol show job %s", jobID)
	output, err := client.RunCommand(cmd)
	out := strings.Trim(output, "\" \t\n\x00")
	if err != nil {
		if strings.Contains(out, invalidJob) {
			return nil, &noJobFound{msg: err.Error()}
		}
		return nil, errors.Wrap(err, out)
	}
	if out != "" {
		return parseJobInfo(strings.NewReader(out))
	}
	return nil, &noJobFound{msg: fmt.Sprintf("no information found for job with id:%q", jobID)}
}

func quoteArgs(t []string) string {
	var args string
	for _, v := range t {
		if !strings.HasPrefix(v, "'") && !strings.HasSuffix(v, "'") {
			v = strings.Replace(v, "'", "\"", -1)
			v = "'" + v + "'"
		}
		args += v + " "
	}
	return args
}

// Convert scalar-unit size to Kib as K for Slurm
func toSlurmMemFormat(memStr string) (string, error) {
	mem, err := humanize.ParseBytes(memStr)
	if err != nil {
		return "", errors.Wrapf(err, "unable to convert to slurm memory format value:%q", memStr)
	}

	// Pass to KiB as K for Slurm
	return strconv.Itoa(int(mem)/1024) + "K", nil
}
