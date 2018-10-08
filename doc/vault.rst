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

Integrate Yorc with a Vault
============================

A Vault is used to store secrets in a secured way.

Yorc allows to interact with a Vault to retrieve sensitive data linked to infrastructures such as 
passwords. 

Currently Yorc supports only `Vault from HashiCorp <https://www.vaultproject.io/>`_ we plan to
support others implementations in Yorc either builtin or by plugins.

The vault integration allows to specify infrastructures parameters as `Go Template <https://golang.org/pkg/text/template/>`_ format and to use
a specific function called ``secret`` this function takes one argument that refers to the secret identifier and an optional list of string arguments
whose signification is dependent to the Vault implementation. This function returns an object implementing the vault.Secret interface which has two
functions ``String()`` that returns the string representation of a secret and ``Raw()`` that returns a Vault implementation-dependent object. The 
second way most powerful but you should look at the Vault implementation documentation to know how to use it.

HashiCorp's Vault integration
------------------------------

HashiCorp's Vault integration is builtin Yorc. Please refer to :ref:`the HashiCorps Vault configuration <option_hashivault>` section to know how to
setup a connection to a running Vault. For more information about Vault itself please refer to its `online documentation <https://www.vaultproject.io/>`_.

Here is how the ``secret`` function is handled by this implementation, the usage is:

``secret "/secret/path/in/vault" ["options" ...]``

Recognized options are:

  * ``data=targetdata``:  Vault allows to store multiple keys/values within a map called `Data`, this option allows to render only the kay named ``targetdata``. Only one data option is allowed. 


The ``String()`` function on the returned secret will render the whole map if there is no ``data`` options specified.

The ``Raw()`` function on the returned secret will return a `github.com/hashicorp/vault/api.Secret <https://godoc.org/github.com/hashicorp/vault/api#Secret>`_.

Bellow are some of the most common ways to get a specific secret using the templating language:

  * ``{{ with (secret "/secret/yorc/mysecret").Raw }}{{ .Data.myKey }}{{end}}``
  * ``{{ secret "/secret/yorc/mysecret" "data=myKey" | print }}``
  * ``{{ (secret "/secret/yorc/mysecret" "data=myKey").String }}``
