package slurm

import (
	"fmt"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/sshutil"
	"strings"
)

// getAttribute allows to return an attribute with defined key from specific treatment
func getAttribute(s sshutil.Session, key string, jobID, nodeName string) (string, error) {
	switch key {
	case "cuda_visible_devices":
		if jobID != "" {
			cmd := fmt.Sprintf("srun --jobid=%s env|grep CUDA_VISIBLE_DEVICES", jobID)
			stdout, err := s.RunCommand(cmd)
			if err != nil {
				return "", errors.Wrapf(err, "Unable to retrieve (%s) for node:%s", key, nodeName)
			}
			value, err := getEnvValue(stdout)
			if err != nil {
				return "", errors.Wrapf(err, "Unable to retrieve (%s) for node:%s", key, nodeName)
			}
			return value, nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unknown key:%s", key)
	}
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
