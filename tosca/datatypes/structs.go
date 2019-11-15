// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package datatypes

//go:generate tdt2go -b -f struct_normative.go ../../data/tosca/normative-types.yml
//go:generate tdt2go -f struct_yorc.go ../../data/tosca/yorc-types.yml
//go:generate tdt2go -f struct_aws.go ../../data/tosca/yorc-aws-types.yml
//go:generate tdt2go -m yorc\.datatypes\.google\.(.*)=Google_$DOLLAR{1} -f struct_google.go ../../data/tosca/yorc-google-types.yml
//go:generate tdt2go -f struct_hostspool.go ../../data/tosca/yorc-hostspool-types.yml
//go:generate tdt2go -f struct_kubernetes.go ../../data/tosca/yorc-kubernetes-types.yml
//go:generate tdt2go -m yorc\.datatypes\.openstack\.(.*)=OS_$DOLLAR{1} -f struct_openstack.go ../../data/tosca/yorc-openstack-types.yml
//go:generate tdt2go -m yorc\.datatypes\.slurm\.(.*)=Slurm_$DOLLAR{1} -f struct_slurm.go ../../data/tosca/yorc-slurm-types.yml
