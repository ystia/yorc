package deployments

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"path"
	"strings"
)

type Resolver struct {
	kv           *api.KV
	deploymentId string
	nodePath     string
	nodeTypePath string
}

func NewResolver(kv *api.KV, deploymentId, nodePath, nodeTypePath string) *Resolver {
	return &Resolver{kv: kv, deploymentId: deploymentId, nodePath: nodePath, nodeTypePath: nodeTypePath}
}

func (r *Resolver) ResolveToscaFunction(function, nodePath, nodeTypePath string, params []string) (string, error) {

	kvPair, _, err := r.kv.Get(nodePath+"/"+function+"/"+params[1], nil)
	if err != nil {
		return "", err
	}
	if kvPair == nil {
		// Look for a default in node type
		// TODO deal with type inheritance
		kvPair, _, err = r.kv.Get(nodeTypePath+"/"+function+"/"+params[1]+"/default", nil)
		if err != nil {
			return "", err
		}
		if kvPair == nil || string(kvPair.Value) == "" {
			return "", fmt.Errorf("Can't retrieve %s %q for type %q either in node definition or node type default", function, params[1], params[0])
		}
	}
	return string(kvPair.Value), nil
}

func (r *Resolver) ResolveExpression(node *tosca.TreeNode) (string, error) {
	log.Debugf("Resolving expression %q", node.Value)
	if node.IsLiteral() {
		return node.Value, nil
	}
	params := make([]string, 0)
	for _, child := range node.Children() {
		exp, err := r.ResolveExpression(child)
		if err != nil {
			return "", err
		}
		params = append(params, exp)
	}
	switch node.Value {
	case "get_property":
		if len(params) != 2 {
			return "", fmt.Errorf("get_property on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			return r.ResolveToscaFunction("properties", r.nodePath, r.nodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_property on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(DeploymentKVPrefix, r.deploymentId, "topology/nodes", params[0])
			kvPair, _, err := r.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], r.deploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(DeploymentKVPrefix, r.deploymentId, "topology/types", nodeType)
			return r.ResolveToscaFunction("properties", nodePath, nodeTypePath, params)
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", fmt.Errorf("get_attribute on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			return r.ResolveToscaFunction("attributes", r.nodePath, r.nodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_attribute on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(DeploymentKVPrefix, r.deploymentId, "topology/nodes", params[0])
			kvPair, _, err := r.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], r.deploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(DeploymentKVPrefix, r.deploymentId, "topology/types", nodeType)
			return r.ResolveToscaFunction("attributes", nodePath, nodeTypePath, params)
		}
	case "concat":
		return strings.Join(params, ""), nil
	}
	return "", fmt.Errorf("Can't resolve expression %q", node.Value)
}
