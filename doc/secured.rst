Run Janus in Secured mode
=========================

To run Janus in secured mode, the following issues have to be addressed:

* Setup a secured Consul cluster
* Setup a secured OpenStack cloud
* Setup a secured Janus server and configure it to use a secured Consul client and the secured OpenStack
* Setup Alien4Cloud security and configure it to use the secured Janus server

To secure the components listed above, and enable TLS, Multi-Domain (SAN) certificates need to be generated.
A short list of commands based on openSSL is provided below.

Generate SSL certificates with SAN
----------------------------------
The SSL certificates you will generate need to be signed by a Certificate Authority.
You might already have one, otherwise, create it using OpenSSL commands below:

.. parsed-literal::

    openssl genrsa -aes256 -out ca.key 4096
    openssl req -new -x509 -days 365 -key ca.key -sha256 -out ca.pem

Generate certificates signed by your CA
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
You need to generate certificates for all the software component to be secured (Consul, Janus, Alien4Cloud).

Use the commands below for each component instance (where <FIP> represents host's IP address):

.. parsed-literal::

    openssl genrsa -out comp.key 4096
    openssl req -new -sha256 -key comp.key  -subj "/C=FR/O=Atos/CN=127.0.0.1" -reqexts SAN -config <(cat /etc/pki/tls/openssl.cnf <(printf "[SAN]\nsubjectAltName=IP:127.0.0.1,IP:<FIP>,DNS:localhost")) -out comp.csr
    openssl x509 -req -in comp.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out comp.pem -days 2048 -extensions SAN -extfile <(cat /etc/pki/tls/openssl.cnf <(printf "[SAN]\nsubjectAltName=IP:127.0.0.1,IP:<FIP>,DNS:localhost"))

In the sections below, the ``comp.key`` and ``comp.pem`` files are used to define the different components' configration.

Secured Consul cluster Setup
----------------------------
Create a ``consul.key`` and ``consul.pem`` for all the Consul agents within the Consul cluster you setup:

 * the server (you may need 3 servers for HA),
 * and the client (you need one client on each host where a Janus server is running).

Use the above commands and replace <FIP> by the host's IP address.

Check Consul documentation for detailes about `agent's configuration <https://www.consul.io/docs/agent/options.html>`_ and `network trafic encryption <https://www.consul.io/docs/agent/encryption.html>`_.

You may find below a typical configuration file for a consul server:

.. code-block:: json

    {
      "domain": "starlings",
      "data_dir": "/tmp/work",
      "client_addr": "0.0.0.0",
      "advertise_addr": "{SERVER_FIP}",
      "server": true,
      "bootstrap": true,
      "encrypt": "{ENCRYPT_KEY}",
      "ports": {
        "https": 8080
      },
      "key_file": "{PATH_TO_CONSUL_SERVER_KEY}",
      "cert_file": "{PATH_TO_CONSUL_SERVER_PEM}",
      "ca_file": "{PATH_TO_CA_PEM}",
      "verify_incoming": true,
      "verify_outgoing": true
    }

And below, one for a consul client.

.. code-block:: json

    {
      "domain": "starlings",
      "data_dir": "/tmp/work",
      "client_addr": "0.0.0.0",
      "advertise_addr": "{FIP}",
      "ui": true,
      "retry_join": [ {SERVER_FIP} ],
      "encrypt": "{ENCRYPT_KEY}",
      "ports": {
        "https": 8080
      },
      "key_file": "{PATH_TO_CONSUL_CLIENT_KEY}",
      "cert_file": "{PATH_TO_CONSUL_CLIENT_PEM}",
      "ca_file": "{PATH_TO_CA_PEM}",
      "verify_incoming_rpc": true,
      "verify_outgoing": true
    }


You can also consult this `Blog <http://russellsimpkins.blogspot.fr/2015/10/consul-adding-tls-using-self-signed.html>`_. You may found unsefull information about how to install CA cerificate in the OS, in case you get errors about trusting the signing authority.

Secured OpenStack 
-----------------
TODO

Secured Janus Setup
-------------------
Create a ``janus-server.key`` and ``janus-server.pem`` using the above commands and replace <FIP> by the host's IP address.

Bellow is an example of configuration file with TLS enabled and using the collocated and secured Consul client.

.. code-block:: JSON

    {
        "os_auth_url": "https://your-openstack:{OPENSTACK_PORT}/v2.0",
        "os_tenant_name": "your-tenant",
        "os_user_name": "os-user",
        "os_password": "os-password",
        "os_prefix": "janus1-",
        "os_private_network_name": "default-private-network",
        "os_default_security_groups": ["default"],
        "key_file": "{PATH_TO_JANUS_SERVER_KEY}",
        "cert_file": "{PATH_TO_JANUS_SERVER_PEM}",
        "consul_ssl": "true",
        "consul_ca_cert": "{PATH_TO_CA_PEM}",
        "consul_key_file": "{PATH_TO_CONSUL_CLIENT_KEY}",
        "consul_cert_file": "{PATH_TO_CONSUL_CLIENT_PEM}",
        "consul_address": "127.0.0.1:8080"
    }

As for Consul, you may need to install CA cerificate in the OS, in case you get errors about trusting the signing authority.

Setup Alien4Cloud security
--------------------------
TODO -See the corresponding Chapter in Alien4Cloud plugin documentation






