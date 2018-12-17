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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/log"
	"golang.org/x/crypto/ssh"
	"io"
	"regexp"
	"strconv"
	"strings"
)

const reSbatch = `^Submitted batch job (\d+)`
const reOutput = `--output=(\w+.*\w+)|-o (\w+.*\w+ )`
const reOutputSBATCH = `^#SBATCH --output=(\w+.*\w+)|^#SBATCH -o (\w+.*\w+ )`

// GetSSHClient returns a SSH client with slurm configuration credentials usage
func GetSSHClient(cfg config.Configuration) (*sshutil.SSHClient, error) {
	// Check slurm configuration
	if err := checkInfraConfig(cfg); err != nil {
		log.Printf("Unable to provide SSH client due to:%+v", err)
		return nil, err
	}

	// Get SSH client
	SSHConfig := &ssh.ClientConfig{
		User:            cfg.Infrastructures[infrastructureName].GetString("user_name"),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Set an authentication method. At least one authentication method
	// has to be set, private/public key or password.
	// The function checkInfraConfig called above ensures at least one of the
	// configuration options, private_key or password, has been defined.
	privateKey := cfg.Infrastructures[infrastructureName].GetString("private_key")
	if privateKey != "" {
		keyAuth, err := sshutil.ReadPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		SSHConfig.Auth = append(SSHConfig.Auth, keyAuth)
	}

	password := cfg.Infrastructures[infrastructureName].GetString("password")
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

// checkInfraConfig checks infrastructure mandatory configuration parameters
func checkInfraConfig(cfg config.Configuration) error {
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

	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("url"), "") == "" {
		return errors.New("slurm infrastructure url is not set")
	}

	if strings.Trim(cfg.Infrastructures[infrastructureName].GetString("port"), "") == "" {
		return errors.New("slurm infrastructure port is not set")
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

func parseJobIDFromBatchOutput(out string) (string, error) {
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

func parseOutputConfigFromBatchScript(r io.Reader, all bool) ([]string, error) {
	res := make([]string, 0)
	scanner := bufio.NewScanner(r)
	reOutput := regexp.MustCompile(reOutput)
	reOutputSBATCH := regexp.MustCompile(reOutputSBATCH)
	for scanner.Scan() {
		line := scanner.Text()
		if all {
			if reOutput.MatchString(line) {
				subMatch := reOutput.FindStringSubmatch(line)
				if subMatch != nil && len(subMatch) == 3 {
					if subMatch[1] != "" {
						res = append(res, subMatch[1])
					} else if subMatch[2] != "" {
						res = append(res, strings.Trim(subMatch[2], " "))
					}
				}
			}
		} else {
			if reOutput.MatchString(line) && !reOutputSBATCH.MatchString(line) {
				subMatch := reOutput.FindStringSubmatch(line)
				if subMatch != nil && len(subMatch) == 3 {
					if subMatch[1] != "" {
						res = append(res, subMatch[1])
					} else if subMatch[2] != "" {
						res = append(res, strings.Trim(subMatch[2], " "))
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return res, err
	}
	return res, nil
}

func parseOutputConfigFromOpts(opts []string) []string {
	res := make([]string, 0)
	for _, opt := range opts {
		propVal := strings.Split(opt, "=")
		if propVal[0] == "--output" {
			res = append(res, propVal[1])
		}
	}
	return res
}

func parseKeyValue(str string) (bool, string, string) {
	keyVal := strings.Split(str, "=")
	if len(keyVal) == 2 && keyVal[0] != "" && keyVal[1] != "" {
		return true, keyVal[0], keyVal[1]
	}
	return false, "", ""
}

func getJobInfo(client sshutil.Client, jobID, jobName string) (*jobInfoShort, error) {
	var cmd string
	if jobID != "" {
		cmd = fmt.Sprintf("squeue --noheader --job=%s -o \"%%j,%%A,%%T\"", jobID)
	} else if jobName != "" {
		cmd = fmt.Sprintf("squeue --noheader --name=%s -o \"%%j,%%A,%%T\"", jobName)
	}

	if cmd == "" {
		return nil, errors.New("getJobInfo needs jobID or jobName parameters to be run")
	}

	output, err := client.RunCommand(cmd)
	if err != nil {
		return nil, errors.Wrap(err, output)
	}
	out := strings.Trim(output, "\" \t\n\x00")
	if out != "" {
		d := strings.Split(out, ",")
		if len(d) != 3 {
			log.Debugf("Unexpected format job information:%q", out)
			return nil, errors.Errorf("Unexpected format:%q for command:%q", cmd, out)
		}
		info := &jobInfoShort{name: d[0], ID: d[1], state: d[2]}
		return info, nil
	}
	return nil, &noJobFound{msg: fmt.Sprintf("no information found for job with id:%q, name:%q", jobID, jobName)}
}
