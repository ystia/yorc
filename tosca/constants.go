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

package tosca

// RunnableInterfaceName is the fully qualified name of the Runnable interface
const RunnableInterfaceName = "tosca.interfaces.node.lifecycle.runnable"

// RunnableSubmitOperationName is the fully qualified name of the Submit operation
const RunnableSubmitOperationName = RunnableInterfaceName + ".submit"

// RunnableRunOperationName is the fully qualified name of the Run operation
const RunnableRunOperationName = RunnableInterfaceName + ".run"

// RunnableCancelOperationName is the fully qualified name of the Cancel operation
const RunnableCancelOperationName = RunnableInterfaceName + ".cancel"

// StandardInterfaceName is the fully qualified name of the Standard interface
const StandardInterfaceName = "tosca.interfaces.node.lifecycle.standard"

// StandardInterfaceShortName is the short name of the Standard interface
const StandardInterfaceShortName = "standard"

// ConfigureInterfaceName is the fully qualified name of the Configure interface
const ConfigureInterfaceName = "tosca.interfaces.relationships.configure"

// ConfigureInterfaceShortName is the short name of the Configure interface
const ConfigureInterfaceShortName = "configure"

// EndpointCapabilityIPAddressAttribute is the name of the attribute containing the ip_address or
// DNS name of tosca.capabilities.Endpoint
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_TYPE_CAPABILITIES_ENDPOINT
const EndpointCapabilityIPAddressAttribute = "ip_address"

// EndpointCapabilityPortProperty is the name of the property containing the port
// of the endpoint
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_TYPE_CAPABILITIES_ENDPOINT
const EndpointCapabilityPortProperty = "port"

// ComputeNodeEndpointCapabilityName is the name of the administrator network
// endpoint capability of a Compute Node
const ComputeNodeEndpointCapabilityName = "endpoint"

// ComputeNodePublicAddressAttributeName is the attribute name of the primary
// public IP address assigned to a Compute Node
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#_Toc379455082
const ComputeNodePublicAddressAttributeName = "public_address"

// ComputeNodePrivateAddressAttributeName is the attribute name of the primary
// private IP address assigned to a Compute Node
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#_Toc379455082
const ComputeNodePrivateAddressAttributeName = "private_address"

// ComputeNodeNetworksAttributeName is the attribute name of the list of
// logical networks assigned to a Compute Node
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#_Toc379455082
const ComputeNodeNetworksAttributeName = "networks"

// MetadataLocationNameKey is the node template metadata key whose value provides
// the name of the location where to create this node template
const MetadataLocationNameKey = "location"

// MetadataTokenKey is the node template metadata key whose value provides
// the value of a token
const MetadataTokenKey = "token"

// MetadataApplicationCredentialIDKey is the node template metadata key whose value provides
// an application credential identifier
const MetadataApplicationCredentialIDKey = "application_credential_id"

// MetadataApplicationCredentialSecretKey is the node template metadata key whose value provides
// the secret value associated to an application credential identifier
const MetadataApplicationCredentialSecretKey = "application_credential_secret"

// NetworkNameProperty is the name of the property containing the name of a logical
// network
//
// See https://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/os/TOSCA-Simple-Profile-YAML-v1.2-os.html#TYPE_TOSCA_DATA_NETWORKINFO
const NetworkNameProperty = "network_name"

// NetworkIDProperty is the name of the property containing the unique ID of a
// network
//
// See https://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/os/TOSCA-Simple-Profile-YAML-v1.2-os.html#TYPE_TOSCA_DATA_NETWORKINFO
const NetworkIDProperty = "network_id"

// NetworkAddressesProperty is the name of the property containing the list of
// IP addresses assigned from the underlying network
//
// See https://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/os/TOSCA-Simple-Profile-YAML-v1.2-os.html#TYPE_TOSCA_DATA_NETWORKINFO
const NetworkAddressesProperty = "addresses"

const (
	// Self is a Tosca keyword which indicates the related object is being referenced from the node itself.
	Self = "SELF"
	// Source is a Tosca keyword which indicates the related object is being referenced from the source relationship.
	Source = "SOURCE"
	// Target is a Tosca keyword which indicates the object is being referenced from the target relationship.
	Target = "TARGET"
	// Host is a Tosca keyword which indicates the object is being referenced from the host in an hostedOn relationship.
	Host = "HOST"
)
