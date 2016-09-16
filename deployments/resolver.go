package deployments

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"strings"
)

type Resolver struct {
	kv           *api.KV
	deploymentId string
}

func NewResolver(kv *api.KV, deploymentId string) *Resolver {
	return &Resolver{kv: kv, deploymentId: deploymentId}
}

// ResolveExpressionForNode resolves a TOSCA expression for a given node.
//
// nodeName is the Node hosting this expression, instanceName is the instance against the expression should be resolved
// this is useful for get_attributes as it may be different for different instances (a classic use case would be 'get_attribute: [ SELF, ip_address ]'
// If you are using it in a context where the node doesn't have multiple instances then instanceName should be an empty string
func (r *Resolver) ResolveExpressionForNode(expression *tosca.TreeNode, nodeName, instanceName string) (string, error) {
	log.Debugf("Resolving expression %q", expression.String())
	if expression.IsLiteral() {
		return expression.Value, nil
	}
	params := make([]string, 0)
	for _, child := range expression.Children() {
		exp, err := r.ResolveExpressionForNode(child, nodeName, instanceName)
		if err != nil {
			return "", err
		}
		params = append(params, exp)
	}

	switch expression.Value {
	case "get_property":
		if len(params) != 2 {
			return "", fmt.Errorf("get_property on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, nodeName, params[1], false)
			return result, err
		case "HOST":
			hostNode, err := GetHostedOnNode(r.kv, r.deploymentId, nodeName)
			if err != nil {
				return "", err
			} else if hostNode == "" {
				// Try to resolve on current node
				hostNode = nodeName
			}
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, hostNode, params[1], true)
			return result, err
		case "SOURCE", "TARGET":
			return "", fmt.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", params[0])
		default:
			// Then it is the name of a modelable entity
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, params[0], params[1], true)
			return result, err
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", fmt.Errorf("get_attribute on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, nodeName, params[1], false)
			if err != nil {
				return "", err
			}
			if !found {
				return "", nil
			}
			return result[instanceName], nil
		case "HOST":
			hostNode, err := GetHostedOnNode(r.kv, r.deploymentId, nodeName)
			if err != nil {
				return "", err
			} else if hostNode == "" {
				// Try to resolve on current node
				hostNode = nodeName
			}
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, hostNode, params[1], false)
			if err != nil {
				return "", err
			}
			if !found {
				return "", nil
			}
			return result[instanceName], nil
		case "SOURCE", "TARGET":
			return "", fmt.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", params[0])
		default:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, params[0], params[1], true)
			if err != nil {
				return "", err
			}
			if !found {
				return "", nil
			}
			return result[instanceName], nil
		}
	case "concat":
		return strings.Join(params, ""), nil
	}
	return "", fmt.Errorf("Can't resolve expression %q", expression.Value)
}

// ResolveExpressionForRelationship resolves a TOSCA expression for a relationship between sourceNode and targetNode.
//
// sourceNode is the Node hosting this expression, instanceName is the instance against the expression should be resolved
// this is useful for get_attributes as it may be different for different instances (a classic use case would be 'get_attribute: [ TARGET, ip_address ]'
// If you are using it in a context where the node doesn't have multiple instances then instanceName should be an empty string
// It returns true as first return param if the expression is in the 'target' context (typically get_attribute: [ TARGET, ip_address ])
func (r *Resolver) ResolveExpressionForRelationship(expression *tosca.TreeNode, sourceNode, targetNode, relationshipType, instanceName string) (bool, string, error) {
	log.Debugf("Resolving expression %q", expression.String())
	if expression.IsLiteral() {
		return false, expression.Value, nil
	}
	params := make([]string, 0)
	for _, child := range expression.Children() {
		_, exp, err := r.ResolveExpressionForRelationship(child, sourceNode, targetNode, relationshipType, instanceName)
		if err != nil {
			return false, "", err
		}
		params = append(params, exp)
	}

	switch expression.Value {
	case "get_property":
		if len(params) != 2 {
			return false, "", fmt.Errorf("get_property on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			_, result, err := GetTypeDefaultProperty(r.kv, r.deploymentId, relationshipType, params[1])
			return false, result, err
		case "HOST":
			return false, "", fmt.Errorf("Keyword %q not supported for a relationship expression", params[0])
		case "SOURCE":
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, sourceNode, params[1], true)
			return false, result, err
		case "TARGET":
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, targetNode, params[1], true)
			return true, result, err
		default:
			// Then it is the name of a modelable entity
			_, result, err := GetNodeProperty(r.kv, r.deploymentId, params[0], params[1], true)
			return false, result, err
		}
	case "get_attribute":
		if len(params) != 2 {
			return false, "", fmt.Errorf("get_attribute on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			_, result, err := GetTypeDefaultAttribute(r.kv, r.deploymentId, relationshipType, params[1])
			return false, result, err
		case "HOST":
			return false, "", fmt.Errorf("Keyword %q not supported for a relationship expression", params[0])
		case "SOURCE":
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, sourceNode, params[1], true)
			if err != nil {
				return false, "", err
			}
			if !found {
				return false, "", nil
			}
			return false, result[instanceName], nil
		case "TARGET":
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, targetNode, params[1], true)
			if err != nil {
				return false, "", err
			}
			if !found {
				return true, "", nil
			}
			return true, result[instanceName], nil
		default:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentId, params[0], params[1], true)
			if err != nil {
				return false, "", err
			}
			if !found {
				return false, "", nil
			}
			return false, result[instanceName], nil
		}
	case "concat":
		return false, strings.Join(params, ""), nil
	}
	return false, "", fmt.Errorf("Can't resolve expression %q", expression.Value)
}
