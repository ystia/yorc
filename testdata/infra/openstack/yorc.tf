resource "openstack_networking_floatingip_v2" "yorc-admin-fp-yorc" {
  count  = "${var.yorc_instances}"
  region = "${var.region}"
  pool   = "${var.public_network_name}"

  depends_on = [
    "openstack_networking_router_interface_v2.yorc-admin-router-port",
  ]
}

# Yorc standard install
data "template_file" "yorc-server-config" {
  count    = "${var.yorc_instances}"
  template = "${file("../config/yorc.config.json.tpl")}"

  vars {
    ks_url      = "${var.keystone_url}"
    ks_tenant   = "${var.keystone_tenant}"
    ks_user     = "${var.keystone_user}"
    ks_password = "${var.keystone_password}"
    region      = "${var.region}"
    priv_net    = "${openstack_networking_network_v2.yorc-admin-net.name}"
    prefix      = "${var.prefix}yorc"
    secgrp      = "${openstack_compute_secgroup_v2.yorc-admin-secgroup.name}"
    statsd_ip   = "${openstack_compute_instance_v2.yorc-monitoring-server.network.0.fixed_ip_v4}"
    server_id     = "${count.index}"
  }
}

data "template_file" "yorc-server-service" {
  template = "${file("../config/yorc.service.tpl")}"

  vars {
    user = "${var.ssh_manager_user}"
  }
}

data "template_file" "yorc-autofs" {
  template = "${file("../config/yorc-autofs.tpl")}"

  vars {
    user          = "${var.ssh_manager_user}"
    nfs_server_ip = "${openstack_compute_instance_v2.nfs-server.network.0.fixed_ip_v4}"
  }
}

data "template_file" "consul-agent-config" {
  count    = "${var.yorc_instances}"
  template = "${file("../config/consul-agent.config.json.tpl")}"

  vars {
    ip_address     = "${element(openstack_compute_instance_v2.yorc-server.*.network.0.fixed_ip_v4, count.index)}"
    consul_servers = "${jsonencode(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4)}"
    statsd_ip      = "${openstack_compute_instance_v2.yorc-monitoring-server.network.0.fixed_ip_v4}"
    consul_ui      = "false"
  }
}

resource "openstack_compute_instance_v2" "yorc-server" {
  count           = "${var.yorc_instances}"
  region          = "${var.region}"
  name            = "${var.prefix}yorc-server-${count.index}"
  image_id        = "${var.yorc_compute_image_id}"
  flavor_id       = "${var.yorc_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.yorc.name}"
  security_groups = ["${openstack_compute_secgroup_v2.yorc-admin-secgroup.name}"]

  availability_zone = "${var.yorc_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.yorc-admin-net.id}"
  }
}

resource "openstack_compute_floatingip_associate_v2" "yorc-server-fip" {
  count       = "${var.yorc_instances}"
  floating_ip = "${element(openstack_networking_floatingip_v2.yorc-admin-fp-yorc.*.address, count.index)}"
  instance_id = "${element(openstack_compute_instance_v2.yorc-server.*.id, count.index)}"
}

resource "null_resource" "yorc-server-provisioning" {
  count = "${var.yorc_instances}"

  connection {
    user        = "${var.ssh_manager_user}"
    host        = "${element(openstack_compute_floatingip_associate_v2.yorc-server-fip.*.floating_ip, count.index)}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "${var.ssh_key_file}"
    destination = "${var.ssh_key_file}"
  }

  provisioner "file" {
    source      = "../yorc"
    destination = "/tmp/yorc"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "file" {
    content     = "${data.template_file.yorc-server-config.*.rendered[count.index]}"
    destination = "/tmp/config.yorc.json"
  }

  provisioner "file" {
    content     = "${data.template_file.yorc-autofs.rendered}"
    destination = "/tmp/auto.yorc"
  }

  provisioner "file" {
    content     = "${data.template_file.consul-agent-config.*.rendered[count.index]}"
    destination = "/tmp/consul-agent.config.json"
  }

  provisioner "file" {
    content     = "${data.template_file.yorc-server-service.rendered}"
    destination = "/tmp/yorc.service"
  }

  provisioner "remote-exec" {
    script = "../scripts/install_consul.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mv /tmp/consul-agent.config.json /etc/consul.d/",
      "sudo chown root:root /etc/consul.d/*",
      "mkdir -p ~/work",
      "chmod 400 ${var.ssh_key_file}",
      "sudo mv /tmp/yorc /usr/local/bin && sudo chmod +x /usr/local/bin/yorc && sudo chown root:root /usr/local/bin/yorc",
      "sudo mv /tmp/yorc.service /etc/systemd/system/yorc.service",
      "sudo chown root:root /etc/systemd/system/yorc.service",
      "sudo yum install -q -y python2-pip nfs-utils autofs",
      "cd /tmp && wget -q https://releases.hashicorp.com/terraform/0.9.11/terraform_0.9.11_linux_amd64.zip && sudo unzip /tmp/terraform_0.9.11_linux_amd64.zip -d /usr/local/bin",
      "sudo -H pip install -q pip --upgrade",
      "sudo -H pip install -q ansible==2.10.0",
      "mv /tmp/config.yorc.json ~/config.yorc.json",
      "echo -e '/-    /etc/auto.direct\n' | sudo tee /etc/auto.master > /dev/null",
      "cat /tmp/auto.yorc | sudo tee /etc/auto.direct > /dev/null",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service yorc.service autofs.service",
      "sudo systemctl restart autofs.service",
      "sudo chown ${var.ssh_manager_user}:${var.ssh_manager_user} ~/work",
      "sudo systemctl start consul.service yorc.service",
    ]
  }
}

output "yorc_addresses" {
  value = ["${openstack_compute_floatingip_associate_v2.yorc-server-fip.*.floating_ip}"]
}
