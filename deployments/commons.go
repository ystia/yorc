package deployments

import (
	"encoding/json"
	"net/url"
	"path"
	"sort"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"vbom.ml/util/sortorder"
)

func urlEscapeAll(keys []string) []string {
	t := make([]string, len(keys))
	for i, k := range keys {
		t[i] = url.QueryEscape(k)
	}
	return t
}

func getValueAssignmentWithoutResolve(kv *api.KV, deploymentID, vaPath string, nestedKeys ...string) (bool, string, bool, error) {
	keyPath := path.Join(vaPath, path.Join(urlEscapeAll(nestedKeys)...))
	kvp, _, err := kv.Get(keyPath, nil)
	if err != nil {
		return false, "", false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		vat := tosca.ValueAssignmentType(kvp.Flags)
		if err != nil {
			return false, "", false, err
		}

		switch vat {
		case tosca.ValueAssignmentLiteral, tosca.ValueAssignmentFunction:
			return true, string(kvp.Value), vat == tosca.ValueAssignmentFunction, nil
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			res, err := readComplexVA(kv, vat, keyPath)
			if err != nil {
				return false, "", false, err
			}
			j, err := json.Marshal(res)
			if err != nil {
				return false, "", false, errors.Wrapf(err, "Failed to generate JSON representation of the complex value assignment key %q", path.Base(vaPath))
			}
			return true, string(j), false, nil
		}
	}
	return false, "", false, nil
}

func getValueAssignment(kv *api.KV, deploymentID, vaPath, nodeName, instanceName, requirementIndex string, nestedKeys ...string) (bool, string, error) {
	found, value, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, vaPath, nestedKeys...)
	if err != nil || !found || !isFunction {
		return found, value, err
	}
	value, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, instanceName, requirementIndex, value, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	return true, value, nil
}

func readComplexVA(kv *api.KV, vaType tosca.ValueAssignmentType, keyPath string) (interface{}, error) {
	keys, _, err := kv.Keys(keyPath+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	var result interface{}
	if vaType == tosca.ValueAssignmentList {
		result = make([]interface{}, len(keys))
	} else {
		result = make(map[string]interface{}, len(keys))
	}

	sort.Sort(sortorder.Natural(keys))
	for i, k := range keys {
		kvp, _, err := kv.Get(k, nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil {
			// Sounds weird to be nil as we listed it just before but also sounds a good practice to check it
			subKeyType := tosca.ValueAssignmentType(kvp.Flags)
			var sr interface{}
			switch subKeyType {
			case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
				sr, err = readComplexVA(kv, subKeyType, k)
				if err != nil {
					return nil, err
				}
			default:
				sr = string(kvp.Value)
			}
			switch v := result.(type) {
			case []interface{}:
				v[i] = sr
			case map[string]interface{}:
				kr, err := url.QueryUnescape(path.Base(k))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to unescape key: %q", path.Base(k))
				}
				v[kr] = sr
			}
		}
	}
	return result, nil
}
