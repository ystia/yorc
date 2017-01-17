package deployments

import (
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

const funcKeywordSELF string = "SELF"
const funcKeywordHOST string = "HOST"
const funcKeywordSOURCE string = "SOURCE"
const funcKeywordTARGET string = "TARGET"

// Resolver is used to resolve TOSCA functions
type Resolver struct {
	kv           *api.KV
	deploymentID string
	taskID       string
}

// NewResolver creates a Resolver instance
func NewResolver(kv *api.KV, deploymentID string, taskID ...string) *Resolver {
	if len(taskID) != 0 {
		return &Resolver{kv: kv, deploymentID: deploymentID, taskID: taskID[0]}
	}
	return &Resolver{kv: kv, deploymentID: deploymentID}
}

// ResolveExpressionForNode resolves a TOSCA expression for a given node.
//
// nodeName is the Node hosting this expression, instanceName is the instance against the expression should be resolved
// this is useful for get_attributes as it may be different for different instances (a classic use case would be 'get_attribute: [ SELF, ip_address ]'
// If you are using it in a context where the node doesn't have multiple instances then instanceName should be an empty string
func (r *Resolver) ResolveExpressionForNode(expression *tosca.TreeNode, nodeName, instanceName string) (string, error) {
	log.Debugf("Deployment %q, Node %q, instanceName %q: Resolving node expression %q", r.deploymentID, nodeName, instanceName, expression.String())
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
			return "", errors.Errorf("get_property on requirement or capability or in nested property is not yet supported")
		}
		switch params[0] {
		case funcKeywordSELF:
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, nodeName, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, nodeName, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			return r.ResolveExpressionForNode(resultExpr.Expression, nodeName, instanceName)
		case funcKeywordHOST:
			hostNode, err := GetHostedOnNode(r.kv, r.deploymentID, nodeName)
			if err != nil {
				return "", err
			} else if hostNode == "" {
				// Try to resolve on current node
				hostNode = nodeName
			}
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, hostNode, params[1])
			if err != nil {
				return "", err
			} else if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, hostNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			return r.ResolveExpressionForNode(resultExpr.Expression, hostNode, instanceName)
		case funcKeywordSOURCE, funcKeywordTARGET:
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", params[0])
		default:
			// Then it is the name of a modelable entity
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, params[0], params[1])
			if err != nil {
				return "", err
			} else if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, params[0], expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			return r.ResolveExpressionForNode(resultExpr.Expression, params[0], instanceName)
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", errors.Errorf("get_attribute on requirement or capability or in nested property is not yet supported")
		}
		switch params[0] {
		case funcKeywordSELF:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, nodeName, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, nodeName, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if r, ok := result[instanceName]; !ok || r == "" {
				return "", nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result[instanceName]), resultExpr)
			if err != nil {
				return "", err
			}
			return r.ResolveExpressionForNode(resultExpr.Expression, nodeName, instanceName)

		case funcKeywordHOST:
			hostNode, err := GetHostedOnNode(r.kv, r.deploymentID, nodeName)
			if err != nil {
				return "", err
			} else if hostNode == "" {
				// Try to resolve on current node
				hostNode = nodeName
			}
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, hostNode, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, hostNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if r, ok := result[instanceName]; !ok || r == "" {
				return "", nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result[instanceName]), resultExpr)
			if err != nil {
				return "", err
			}
			return r.ResolveExpressionForNode(resultExpr.Expression, hostNode, instanceName)
		case funcKeywordSOURCE, funcKeywordTARGET:
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", params[0])
		default:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, params[0], params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, node %q, can't resolve expression %q", r.deploymentID, params[0], expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if len(result) > 1 {
				log.Printf("Deployment %q, node %q: Expression %q returned multiple (%d) values in a scalar context. A random one will be choose which may lead to unpredicable results.", r.deploymentID, nodeName, expression, len(result))
				events.LogEngineMessage(r.kv, r.deploymentID, fmt.Sprintf("Node %q: Expression %q returned multiple (%d) values in a scalar context. A random one will be choose which may lead to unpredicable results.", nodeName, expression, len(result)))
			}
			for modEntityInstance, modEntityResult := range result {
				// Return during the first processing (cf warning above)
				if modEntityResult == "" {
					return modEntityResult, nil
				}
				resultExpr := &tosca.ValueAssignment{}
				err = yaml.Unmarshal([]byte(modEntityResult), resultExpr)
				if err != nil {
					return "", err
				}
				return r.ResolveExpressionForNode(resultExpr.Expression, params[0], modEntityInstance)
			}
		}
	case "concat":
		return strings.Join(params, ""), nil
	case "get_input":
		if len(params) != 1 {
			return "", errors.Errorf("get_input doesn't support more than one parameter")
		}
		inputVal, err := GetInputValue(r.kv, r.deploymentID, params[0])
		if err != nil {
			return "", err
		} else if inputVal == "" {
			return inputVal, nil
		}
		resultExpr := &tosca.ValueAssignment{}
		err = yaml.Unmarshal([]byte(inputVal), resultExpr)
		if err != nil {
			return "", err
		}
		return r.ResolveExpressionForNode(resultExpr.Expression, nodeName, instanceName)
	}
	return "", errors.Errorf("Can't resolve expression %q", expression.Value)
}

// ResolveExpressionForRelationship resolves a TOSCA expression for a relationship between sourceNode and targetNode.
//
// sourceNode is the Node hosting this expression, instanceName is the instance against the expression should be resolved
// this is useful for get_attributes as it may be different for different instances (a classic use case would be 'get_attribute: [ TARGET, ip_address ]'
// If you are using it in a context where the node doesn't have multiple instances then instanceName should be an empty string
// It returns true as first return param if the expression is in the 'target' context (typically get_attribute: [ TARGET, ip_address ])
func (r *Resolver) ResolveExpressionForRelationship(expression *tosca.TreeNode, sourceNode, targetNode, requirementIndex, instanceName string) (string, error) {
	log.Debugf("Deployment %q, sourceNode %q, targetNode %q, requirement index %q, instanceName %q: Resolving expression %q", r.deploymentID, sourceNode, targetNode, requirementIndex, instanceName, expression.String())
	if expression.IsLiteral() {
		return expression.Value, nil
	}
	params := make([]string, 0)
	for _, child := range expression.Children() {
		exp, err := r.ResolveExpressionForRelationship(child, sourceNode, targetNode, requirementIndex, instanceName)
		if err != nil {
			return "", err
		}
		params = append(params, exp)
	}

	switch expression.Value {
	case "get_property":
		if len(params) != 2 {
			return "", errors.Errorf("get_property on requirement or capability or in nested property is not yet supported")
		}
		switch params[0] {
		case funcKeywordSELF:
			found, result, err := GetRelationshipPropertyFromRequirement(r.kv, r.deploymentID, sourceNode, requirementIndex, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}

			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			result, err = r.ResolveExpressionForRelationship(resultExpr.Expression, sourceNode, targetNode, requirementIndex, instanceName)
			return result, err
		case funcKeywordHOST:
			return "", errors.Errorf("Keyword %q not supported for a relationship expression", params[0])
		case funcKeywordSOURCE:
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, sourceNode, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			result, err = r.ResolveExpressionForNode(resultExpr.Expression, sourceNode, instanceName)
			return result, err
		case funcKeywordTARGET:
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, targetNode, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q, target node %q, can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, targetNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			result, err = r.ResolveExpressionForNode(resultExpr.Expression, targetNode, instanceName)
			return result, err
		default:
			// Then it is the name of a modelable entity
			found, result, err := GetNodeProperty(r.kv, r.deploymentID, params[0], params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, params[0], expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			result, err = r.ResolveExpressionForNode(resultExpr.Expression, params[0], instanceName)
			return result, err
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", errors.Errorf("get_attribute on requirement or capability or in nested property is not yet supported")
		}
		switch params[0] {
		case funcKeywordSELF:
			kvp, _, err := r.kv.Get(path.Join(consulutil.DeploymentKVPrefix, r.deploymentID, "topology/nodes", sourceNode, "requirements", requirementIndex, "relationship"), nil)
			if err != nil {
				return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			if kvp == nil || len(kvp.Value) == 0 {
				return "", errors.Errorf("Deployment %q, requirement index %q, in source node %q can't retrieve relationship type. (Expression was %q)", r.deploymentID, requirementIndex, sourceNode, expression.String())
			}
			relationshipType := string(kvp.Value)
			found, result, err := GetTypeDefaultAttribute(r.kv, r.deploymentID, relationshipType, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if result == "" {
				return result, nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result), resultExpr)
			if err != nil {
				return "", err
			}
			result, err = r.ResolveExpressionForRelationship(resultExpr.Expression, sourceNode, targetNode, requirementIndex, instanceName)
			return result, err
		case funcKeywordHOST:
			return "", errors.Errorf("Keyword %q not supported for a relationship expression", params[0])
		case funcKeywordSOURCE:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, sourceNode, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if r, ok := result[instanceName]; !ok || r == "" {
				return "", nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result[instanceName]), resultExpr)
			if err != nil {
				return "", err
			}
			res, err := r.ResolveExpressionForNode(resultExpr.Expression, sourceNode, instanceName)
			return res, err
		case funcKeywordTARGET:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, targetNode, params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q, target node %q, can't resolve expression %q", r.deploymentID, requirementIndex, sourceNode, targetNode, expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if r, ok := result[instanceName]; !ok || r == "" {
				return "", nil
			}
			resultExpr := &tosca.ValueAssignment{}
			err = yaml.Unmarshal([]byte(result[instanceName]), resultExpr)
			if err != nil {
				return "", err
			}
			res, err := r.ResolveExpressionForNode(resultExpr.Expression, targetNode, instanceName)
			return res, err
		default:
			found, result, err := GetNodeAttributes(r.kv, r.deploymentID, params[0], params[1])
			if err != nil {
				return "", err
			}
			if !found {
				log.Debugf("Deployment %q, requirement index %q, in source node %q can't resolve expression %q", r.deploymentID, requirementIndex, params[0], expression.String())
				return "", errors.Errorf("Can't resolve expression %q", expression.String())
			}
			if len(result) > 1 {
				log.Printf("Deployment %q, SourceNode %q, TargetNode %q, requirement index %q: Expression %q returned multiple (%d) values in a scalar context. A random one will be choose which may lead to unpredicable results.", r.deploymentID, sourceNode, targetNode, requirementIndex, expression, len(result))
				events.LogEngineMessage(r.kv, r.deploymentID, fmt.Sprintf("SourceNode %q, TargetNode %q, requirement index %q: Expression %q returned multiple (%d) values in a scalar context. A random one will be choose which may lead to unpredicable results.", sourceNode, targetNode, requirementIndex, expression, len(result)))
			}
			for modEntityInstance, modEntityResult := range result {
				// Return during the first processing (cf warning above)
				if modEntityResult == "" {
					return modEntityResult, nil
				}
				resultExpr := &tosca.ValueAssignment{}
				err = yaml.Unmarshal([]byte(modEntityResult), resultExpr)
				if err != nil {
					return "", err
				}
				return r.ResolveExpressionForNode(resultExpr.Expression, params[0], modEntityInstance)
			}
		}
	case "concat":
		return strings.Join(params, ""), nil
	case "get_input":
		if len(params) != 1 {
			return "", errors.Errorf("get_input doesn't support more than one parameter")
		}
		inputVal, err := GetInputValue(r.kv, r.deploymentID, params[0])
		if err != nil {
			return "", err
		} else if inputVal == "" {
			return inputVal, nil
		}
		resultExpr := &tosca.ValueAssignment{}
		err = yaml.Unmarshal([]byte(inputVal), resultExpr)
		if err != nil {
			return "", err
		}
		return r.ResolveExpressionForRelationship(resultExpr.Expression, sourceNode, targetNode, requirementIndex, instanceName)
	}
	return "", errors.Errorf("Can't resolve expression %q", expression.Value)
}

// ResolveCustomCommandInput resolve a Custom command input.
func (r *Resolver) ResolveCustomCommandInput(inputName string) (string, error) {
	// TODO: is here the best place for this? It is related to tasks.
	kvP, _, err := r.kv.Get(path.Join(consulutil.TasksPrefix, r.taskID, "inputs", inputName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	return string(kvP.Value), nil
}
