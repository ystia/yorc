package deployments

import (
	"encoding/json"
	"net/url"
	"path"
	"sort"
	"strings"

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

func getValueAssignmentWithoutResolve(kv *api.KV, deploymentID, vaPath, baseDatatype string, nestedKeys ...string) (bool, string, bool, error) {
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
			res, err := readComplexVA(kv, vat, deploymentID, keyPath, baseDatatype, nestedKeys...)
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
	if baseDatatype != "" && !strings.HasPrefix(baseDatatype, "list:") && !strings.HasPrefix(baseDatatype, "map:") && len(nestedKeys) > 0 {
		found, result, isFunc, err := getTypeDefaultProperty(kv, deploymentID, baseDatatype, nestedKeys[0], nestedKeys[1:]...)
		if err != nil {
			return false, "", false, err
		}
		if found {
			return true, result, isFunc, nil
		}
	}
	// not found
	return false, "", false, nil
}

func getValueAssignment(kv *api.KV, deploymentID, vaPath, nodeName, instanceName, requirementIndex string, nestedKeys ...string) (bool, string, error) {
	return getValueAssignmentWithDataType(kv, deploymentID, vaPath, nodeName, instanceName, requirementIndex, "", nestedKeys...)
}

func getValueAssignmentWithDataType(kv *api.KV, deploymentID, vaPath, nodeName, instanceName, requirementIndex, baseDatatype string, nestedKeys ...string) (bool, string, error) {

	found, value, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, vaPath, baseDatatype, nestedKeys...)
	if err != nil || !found || !isFunction {
		return found, value, err
	}
	value, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, instanceName, requirementIndex, value, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	return true, value, nil
}

func readComplexVA(kv *api.KV, vaType tosca.ValueAssignmentType, deploymentID, keyPath, baseDatatype string, nestedKeys ...string) (interface{}, error) {
	keys, _, err := kv.Keys(keyPath+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	var result interface{}
	if vaType == tosca.ValueAssignmentList {
		result = make([]interface{}, 0)
	} else {
		result = make(map[string]interface{})
	}

	sort.Sort(sortorder.Natural(keys))
	var i int
	for _, k := range keys {
		kvp, _, err := kv.Get(k, nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil {
			// Sounds weird to be nil as we listed it just before but also sounds a good practice to check it
			kr, err := url.QueryUnescape(path.Base(k))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unescape key: %q", path.Base(k))
			}
			subKeyType := tosca.ValueAssignmentType(kvp.Flags)
			var sr interface{}
			switch subKeyType {
			case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
				newNestedkeys := append(nestedKeys, kr)
				sr, err = readComplexVA(kv, subKeyType, deploymentID, k, baseDatatype, newNestedkeys...)
				if err != nil {
					return nil, err
				}
			default:
				sr = string(kvp.Value)
			}
			switch v := result.(type) {
			case []interface{}:
				result = append(v, sr)
			case map[string]interface{}:
				v[kr] = sr
			}
			i++
		}
	}

	if baseDatatype != "" && vaType == tosca.ValueAssignmentMap {
		currentDatatype, err := GetNestedDataType(kv, deploymentID, baseDatatype, nestedKeys...)
		if err != nil {
			return nil, err
		}
		castedResult := result.(map[string]interface{})
		for currentDatatype != "" {
			dtProps, err := GetTypeProperties(kv, deploymentID, currentDatatype)
			if err != nil {
				return nil, err
			}
			for _, prop := range dtProps {
				if _, ok := castedResult[prop]; !ok {
					// not found check default
					found, result, _, err := getTypeDefaultProperty(kv, deploymentID, currentDatatype, prop)
					if err != nil {
						return nil, err
					}
					if found {
						castedResult[prop] = result
					}
				}
			}
			currentDatatype, err = GetParentType(kv, deploymentID, currentDatatype)
			if err != nil {
				if IsTypeMissingError(err) {
					// builtin YAML types like string, integer, map, list and so on are not
					// considered as TOSCA types so just stop. This is not an error
					break
				}
				return nil, err
			}
		}
	}
	return result, nil
}
