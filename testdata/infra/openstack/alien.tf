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

  provisioner "remote-exec" {
    inline = [
      "sudo yum install -q -y wget java-1.8.0-openjdk",
      "wget -q -O alien4cloud.tgz \"${var.alien_download_url}\"",
      "mkdir -p ~/alien4cloud",
      "tar xzvf alien4cloud.tgz --strip-components 1 -C ~/alien4cloud",
      "sudo mv /tmp/alien.service /etc/systemd/system/alien.service",
      "sudo chown root:root /etc/systemd/system/alien.service",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable alien.service",
      "sudo systemctl start alien.service",
    ]
  }
}

output "alien4cloud" {
  value = "http://${openstack_compute_floatingip_associate_v2.alien-server-fip.floating_ip}:8088"
}
