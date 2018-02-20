resource "openstack_compute_keypair_v2" "yorc" {
  name       = "${var.prefix}yorc"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

resource "openstack_networking_network_v2" "yorc-admin-net" {
  name           = "${var.prefix}yorc-admin-network"
  admin_state_up = "true"
  region         = "${var.region}"
}

resource "openstack_networking_subnet_v2" "yorc-admin-subnet" {
  name        = "${var.prefix}yorc-admin-subnet"
  network_id  = "${openstack_networking_network_v2.yorc-admin-net.id}"
  cidr        = "10.0.1.0/24"
  ip_version  = 4
  enable_dhcp = "true"

  #  dns_nameservers = ["8.8.8.8","8.8.4.4"]
  region = "${var.region}"
}

resource "openstack_networking_router_v2" "yorc-admin-router" {
  name             = "${var.prefix}yorc-admin-router"
  admin_state_up   = "true"
  external_gateway = "${var.external_gateway}"
  region           = "${var.region}"
}

resource "openstack_networking_router_interface_v2" "yorc-admin-router-port" {
  router_id = "${openstack_networking_router_v2.yorc-admin-router.id}"
  subnet_id = "${openstack_networking_subnet_v2.yorc-admin-subnet.id}"
  region    = "${var.region}"
}

resource "openstack_compute_secgroup_v2" "yorc-admin-secgroup" {
  name        = "${var.prefix}yorc-admin-secgrp"
  description = "Yorc Admin Openbar"

  rule {
    ip_protocol = "tcp"
    from_port   = 1
    to_port     = 65535
    cidr        = "0.0.0.0/0"
  }

  rule {
    ip_protocol = "udp"
    from_port   = 1
    to_port     = 65535
    cidr        = "0.0.0.0/0"
  }

  rule {
    ip_protocol = "icmp"
    from_port   = -1
    to_port     = -1
    cidr        = "0.0.0.0/0"
  }
}
