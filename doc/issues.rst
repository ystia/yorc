..
   Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
   ---

.. _yorc_issues_section:

Known issues
============

.. _yorc_ber_issue_section:

BER for SSH private key is not supported
----------------------------------------

Yorc uses SSH to connect to provisioned Computes or to hosts from hosts pool.

Default behavior is to add related private keys to SSH-agent in order to handle authentication.

But in some cases, SSH-agent can't be used and authentication must be done with private key file with the :ref:`--disable_ssh_agent <option_disable_ssh_agent_cmd>` command-line flag

As we use Golang ssh package (https://godoc.org/golang.org/x/crypto/ssh) to parse the private key, we don't support
BER encoding format (https://github.com/golang/go/issues/14145).
This kind of format is especially used by OpenStack Liberty SSH keypair generator.
