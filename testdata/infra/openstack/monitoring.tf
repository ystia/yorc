resource "openstack_networking_floatingip_v2" "janus-monitoring-fp-janus" {
  region = "${var.region}"
  pool   = "${var.public_network_name}"

  depends_on = [
    "openstack_networking_router_interface_v2.janus-admin-router-port",
  ]
}

resource "openstack_compute_instance_v2" "janus-monitoring-server" {
  region          = "${var.region}"
  name            = "${var.prefix}janus-monitoring"
  image_id        = "${var.janus_compute_image_id}"
  flavor_id       = "${var.janus_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.janus.name}"
  security_groups = ["${openstack_compute_secgroup_v2.janus-admin-secgroup.name}"]

  availability_zone = "${var.janus_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.janus-admin-net.id}"
  }
}

resource "openstack_compute_floatingip_associate_v2" "janus-monitoring-fip" {
  floating_ip = "${openstack_networking_floatingip_v2.janus-monitoring-fp-janus.address}"
  instance_id = "${openstack_compute_instance_v2.janus-monitoring-server.id}"
}

data "template_file" "janus-monitoring-consul-agent-config" {
  template = "${file("../config/consul-agent.config.json.tpl")}"

  vars {
    ip_address     = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
    consul_servers = "${jsonencode(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4)}"
    statsd_ip      = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
    consul_ui      = "true"
  }
}

resource "null_resource" "janus-monitoring-provisioning-install-consul" {
  connection {     
    agent       = false
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    content     = "${data.template_file.janus-monitoring-consul-agent-config.rendered}"
    destination = "/tmp/consul-agent.config.json"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "remote-exec" {
    script = "../scripts/install_consul.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mv /tmp/consul-agent.config.json /etc/consul.d/",
      "sudo chown root:root /etc/consul.d/*",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service",
      "sudo systemctl start consul.service",
    ]
  }
}

resource "null_resource" "janus-monitoring-provisioning-install-docker" {
  depends_on = ["null_resource.janus-monitoring-provisioning-install-consul"]

  connection {     
    agent       = false
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "../config/docker-daemon.json"
    destination = "/tmp/docker-daemon.json"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo yum install -q -y yum-utils",
      "sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo",
      "sudo yum makecache fast -q",
      "sudo yum install -q -y device-mapper-persistent-data lvm2",
      "sudo yum install -q -y docker-ce",
      "sudo mkdir -p /etc/docker/",
      "sudo chown root:root /tmp/docker-daemon.json && sudo mv /tmp/docker-daemon.json /etc/docker/daemon.json",
      "sudo systemctl enable docker",
      "sudo systemctl start docker",
      "sudo usermod -aG docker ${var.ssh_manager_user}",
    ]
  }
}

data "template_file" "janus-monitoring-docker-config" {
  count    = "${var.http_proxy != "" ? 1 : 0 }"
  template = "${file("../config/docker-systemd-proxy.conf.tpl")}"

  vars {
    proxy = "${var.http_proxy}"
  }
}

data "template_file" "prometheus-config" {
  template = "${file("../config/prometheus.yml.tpl")}"

  vars {
    janus_targets = "${join(", ", formatlist("'%s:8800'", openstack_compute_instance_v2.janus-server.*.network.0.fixed_ip_v4))}"
  }
}

resource "null_resource" "janus-monitoring-provisioning-config-docker" {
  depends_on = ["null_resource.janus-monitoring-provisioning-install-docker"]
  count      = "${var.http_proxy != "" ? 1 : 0 }"

  connection {     
    agent       = false
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    content     = "${data.template_file.janus-monitoring-docker-config.rendered}"
    destination = "/tmp/docker-proxy.conf"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p /etc/systemd/system/docker.service.d/",
      "sudo chown root:root /tmp/docker-proxy.conf && sudo mv /tmp/docker-proxy.conf /etc/systemd/system/docker.service.d/proxy.conf",
      "sudo systemctl daemon-reload",
      "sudo systemctl restart docker",
    ]
  }
}

resource "null_resource" "janus-monitoring-provisioning-start-monitoring-statsd-grafana" {
  depends_on = ["null_resource.janus-monitoring-provisioning-config-docker"]

  connection {     
    agent       = false
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "../grafana_dashboards"
    destination = "/tmp"
  }

  provisioner "remote-exec" {
    inline = [
      # | cat is a workaround the lack of --quiet option for docker cli as it is not a tty docker will reduce outputs
      "docker run --restart unless-stopped -d -p 80:80 -p 8125:8125/udp -p 8126:8126 --name kamon-grafana-dashboard kamon/grafana_graphite | cat",

      "while [[ \"$(curl --noproxy 127.0.0.1 -s -I http://admin:admin@127.0.0.1/ | head -n 1|cut -d' ' -f2)\" != \"200\" ]]; do echo 'Waiting for Grafana to finish to startup' && sleep 5; done",
      "set -x",
      "curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/datasources -H 'Content-type: application/json' -X POST -d '{\"Name\":\"graphite\",\"Type\":\"graphite\",\"IsDefault\":true,\"Url\":\"http://localhost:8000\",\"Access\":\"proxy\",\"BasicAuth\":false}'",
      "curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/datasources -H 'Content-type: application/json' -X POST -d '{\"Name\":\"prometheus\",\"Type\":\"prometheus\",\"IsDefault\":false,\"Url\":\"http://${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}:9090\",\"Access\":\"proxy\",\"BasicAuth\":false}'",
      "for g in $$(find /tmp/grafana_dashboards -type f -name '*.json'); do curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/dashboards/db -H 'Content-type: application/json' -X POST -d @$${g}; done",
    ]
  }
}
resource "null_resource" "janus-monitoring-provisioning-start-monitoring-prometheus" {
  depends_on = ["null_resource.janus-monitoring-provisioning-config-docker"]

  connection {     
    agent       = false
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    content     = "${data.template_file.prometheus-config.rendered}"
    destination = "/home/${var.ssh_manager_user}/prometheus.yml"
  }

  provisioner "remote-exec" {
    inline = [
      # | cat is a workaround the lack of --quiet option for docker cli as it is not a tty docker will reduce outputs
      "docker run --restart unless-stopped -d -p 9090:9090 -v /home/${var.ssh_manager_user}/prometheus.yml:/etc/prometheus/prometheus.yml --name prometheus prom/prometheus | cat",
    ]
  }
}

output "janus-monitoring" {
  value = "http://${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
}
