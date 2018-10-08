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

Run Yorc in Secured mode
=========================

To run Yorc in secured mode, the following issues have to be addressed:

* Setup a secured Consul cluster
* Setup a secured Yorc server and configure it to use a secured Consul client
* Setup Alien4Cloud security and configure it to use a secured Yorc server

In the case of Yorc HA setup (see :ref:`yorc_ha_section`), all the Yorc servers composing the cluster need to be secured.

To secure the components listed above, and enable TLS, Multi-Domain (SAN) certificates need to be generated.
A short list of commands based on openSSL is provided below.

Generate SSL certificates with SAN
----------------------------------
The SSL certificates you will generate need to be signed by a Certificate Authority.
You might already have one, otherwise, create it using OpenSSL commands below:

.. code-block:: bash

    openssl genrsa -aes256 -out ca.key 4096
    openssl req -new -x509 -days 365 -key ca.key -sha256 -out ca.pem

Generate certificates signed by your CA
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
You need to generate certificates for all the software component to be secured (Consul agents, Yorc servers, Alien4Cloud).

Use the commands below for each component instance (where <IP> represents IP address used to connect to the component).
Replace ``comp`` by a string of your choise corresponding to the components to be secured.

.. code-block:: bash

    openssl genrsa -out comp.key 4096
    openssl req -new -sha256 -key comp.key  -subj "/C=FR/O=Atos/CN=127.0.0.1" -reqexts SAN -config <(cat /etc/pki/tls/openssl.cnf <(printf "[SAN]\nsubjectAltName=IP:127.0.0.1,IP:<IP>,DNS:localhost")) -out comp.csr
    openssl x509 -req -in comp.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out comp.pem -days 2048 -extensions SAN -extfile <(cat /etc/pki/tls/openssl.cnf <(printf "[SAN]\nsubjectAltName=IP:127.0.0.1,IP:<IP>,DNS:localhost"))

In the sections below, the ``comp.key`` and ``comp.pem`` files path are used in the components' configuration file.

Secured Consul cluster Setup
----------------------------
.. note:: You need to generate cerificates for all the Consul agents within the Consul cluster you setup.

In a High Availability cluster, you need to setup at least 3 consul servers, and one consul client on each host where a Yorc server is running. 

Check Consul documentation for details about `agent's configuration <https://www.consul.io/docs/agent/options.html>`_.

You may find below a typical configuration file for a consul server ; to be updated after having generated the ``consul_server.key`` and ``consul_server.pem`` files.

.. code-block:: json

    {
        "domain": "yorc",
        "data_dir": "/tmp/work",
        "client_addr": "0.0.0.0",
        "advertise_addr": "{SERVER_IP}",
        "server": true,
        "bootstrap": true,
        "ui": true,
        "encrypt": "{ENCRYPT_KEY},
        "ports": {
            "https": 8543
        },
        "key_file": "{PATH_TO_CONSUL_SERVER_KEY}",
        "cert_file": "{PATH_TO_CONSUL_SERVER_PEM}",
        "ca_file": "{PATH_TO_CA_PEM}",
        "verify_incoming": true,
        "verify_outgoing": true
    }

And below, a typical configuration file for a consul client.

.. code-block:: json

    {
      "domain": "yorc",
      "data_dir": "/tmp/work",
      "client_addr": "0.0.0.0",
      "advertise_addr": "{IP}",
      "retry_join": [ "{SERVER_IP}" ],
      "encrypt": "{ENCRYPT_KEY},
      "ports": {
        "https": 8543
      },
      "key_file": "{PATH_TO_CONSUL_CLIENT_KEY}",
      "cert_file": "{PATH_TO_CONSUL_CLIENT_PEM}",
      "ca_file": "{PATH_TO_CA_PEM}",
      "verify_incoming_rpc": true,
      "verify_outgoing": true
    }

In the above example, the encryption is enabled for the gossip traffic inside the Consul cluster. Check Consul documentation for details `network traffic encryption <https://www.consul.io/docs/agent/encryption.html>`_.

You can also consult this `Blog <http://russellsimpkins.blogspot.fr/2015/10/consul-adding-tls-using-self-signed.html>`_. 
You may found useful information about how to install CA certificate in the OS, in case you get errors about trusting the signing authority.

Secured Yorc Setup
------------------

Generate a ``yorc_server.key`` and ``yorc_server.pem`` using the above commands and replace <IP> by the host's IP address.

Bellow is an example of configuration file with TLS enabled and using the collocated and secured Consul client.

.. code-block:: JSON

    {
        "consul": {
            "ssl": "true",
            "ca_cert": "{PATH_TO_CA_PEM}",
            "key_file": "{PATH_TO_CONSUL_CLIENT_KEY}",
            "cert_file": "{PATH_TO_CONSUL_CLIENT_PEM}",
            "address": "127.0.0.1:8543"
        },
        "resources_prefix": "yorc1-",
        "key_file": "{PATH_TO_YORC_SERVER_KEY}",
        "cert_file": "{PATH_TO_YORC_SERVER_PEM}",
        "ssl_verify": true,
        "infrastructures" : {
            "openstack": {
                "auth_url": "https://your-openstack:{OPENSTACK_PORT}/v2.0",
                "tenant_name": "your-tenant",
                "user_name": "os-user",
                "password": "os-password",
                "private_network_name": "default-private-network",
                "default_security_groups": ["default"]
            }
        }
    }

In the above example SSL verification is enabled for Yorc (ssl_verify set to true). In this case, the Consul Agent must be enabled to use TLS configuration files for HTTP health checks. Otherwise, the TLS handshake may fail.
You can find below the Consul agent's configuration:

.. code-block:: json

    {
      "domain": "yorc",
      "data_dir": "/tmp/work",
      "client_addr": "0.0.0.0",
      "advertise_addr": "{IP}",
      "ui": true,
      "retry_join": [ "{SERVER_IP}" ],
      "encrypt": "{ENCRYPT_KEY}",
      "ports": {
        "https": 8543
      },
      "key_file": "{PATH_TO_CONSUL_CLIENT_KEY}",
      "cert_file": "{PATH_TO_CONSUL_CLIENT_PEM}",
      "ca_file": "{PATH_TO_CA_PEM}",
      "enable_agent_tls_for_checks": true,
      "verify_incoming_rpc": true,
      "verify_outgoing": true
    }

As for Consul, you may need to install CA certificate in the OS, in case you get errors about trusting the signing authority.

Secured Yorc CLI Setup
----------------------

If ``ssl_verify`` is enabled for Yorc server, the Yorc CLI have to provide a client certificate signed by the Yorc's Certificate Authority.

So, create a ``yorc_client.key`` and ``yorc_client.pem`` using the above commands and replace <IP> by the host's IP address.

Bellow is an example of configuration file with TLS enabled. Refer to :ref:`yorc_config_client_section` for more information.

.. code-block:: JSON

    {
        "key_file": "{PATH_TO_YORC_CLIENT_KEY}",
        "cert_file": "{PATH_TO_YORC_CLIENT_PEM}",
        "ca_file": "{PATH_TO_CA_PEM}",
        "yorc_api": "<YORC_SERVER_IP>:8800"
    }


Setup Alien4Cloud security
--------------------------

See the corresponding Chapter in Alien4Cloud plugin documentation

