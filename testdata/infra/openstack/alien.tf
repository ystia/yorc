resource "openstack_networking_floatingip_v2" "janus-admin-fp-alien" {
  region = "${var.region}"
  pool   = "${var.public_network_name}"

  depends_on = [
    "openstack_networking_router_interface_v2.janus-admin-router-port",
  ]
}

resource "openstack_compute_instance_v2" "alien-server" {
  region          = "${var.region}"
  name            = "${var.prefix}alien-server"
  image_id        = "${var.janus_compute_image_id}"
  flavor_id       = "${var.janus_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.janus.name}"
  security_groups = ["${openstack_compute_secgroup_v2.janus-admin-secgroup.name}"]

  availability_zone = "${var.janus_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.janus-admin-net.id}"
  }
}

resource "openstack_compute_floatingip_associate_v2" "alien-server-fip" {
  floating_ip = "${element(openstack_networking_floatingip_v2.janus-admin-fp-alien.*.address, count.index)}"
  instance_id = "${openstack_compute_instance_v2.alien-server.id}"
}

data "template_file" "alien-server-service" {
  template = "${file("../config/alien.service.tpl")}"

  vars {
    user = "${var.ssh_manager_user}"
  }
}

data "template_file" "alien-consul-checks" {
  template = "${file("../config/alien-consul-check.json.tpl")}"

  vars {
    ip_address = "${openstack_compute_instance_v2.alien-server.network.0.fixed_ip_v4}"
  }
}

data "template_file" "consul-agent-4alien-config" {
  template = "${file("../config/consul-agent.config.json.tpl")}"

  vars {
    ip_address     = "${openstack_compute_instance_v2.alien-server.network.0.fixed_ip_v4}"
    consul_servers = "${jsonencode(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4)}"
    statsd_ip      = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
    consul_ui      = "false"
  }
}

resource "null_resource" "alien-server-provisioning" {
  connection {
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.alien-server-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    content     = "${data.template_file.alien-server-service.rendered}"
    destination = "/tmp/alien.service"
  }

  provisioner "file" {
    content     = "${data.template_file.consul-agent-4alien-config.rendered}"
    destination = "/tmp/consul-agent.config.json"
  }

  provisioner "file" {
    content     = "${data.template_file.alien-consul-checks.rendered}"
    destination = "/tmp/alien-consul-check.json"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "remote-exec" {
    script = "../scripts/install_dnsmasq.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p /etc/consul.d",
      "sudo mv /tmp/consul-agent.config.json /etc/consul.d/",
      "sudo mv /tmp/alien-consul-check.json /etc/consul.d/",
      "sudo chown root:root /etc/consul.d/*",
      "sudo mv /tmp/consul.service /etc/systemd/system/consul.service",
      "sudo chown root:root /etc/systemd/system/consul.service",
      "sudo yum install -q -y wget java-1.8.0-openjdk zip unzip",
      "cd /tmp && wget -q https://releases.hashicorp.com/consul/0.8.1/consul_0.8.1_linux_amd64.zip && sudo unzip /tmp/consul_0.8.1_linux_amd64.zip -d /usr/local/bin",
      "wget -q -O alien4cloud.tgz \"${var.alien_download_url}\"",
      "mkdir -p ~/alien4cloud",
      "tar xzvf alien4cloud.tgz --strip-components 1 -C ~/alien4cloud",
      "sudo mv /tmp/alien.service /etc/systemd/system/alien.service",
      "sudo chown root:root /etc/systemd/system/alien.service",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service alien.service",
      "sudo systemctl start consul.service alien.service",
    ]
  }
}

output "alien4cloud" {
  value = "http://${openstack_compute_floatingip_associate_v2.alien-server-fip.floating_ip}:8088"
}
