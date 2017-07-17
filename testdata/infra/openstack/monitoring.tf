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

resource "null_resource" "janus-montoring-provisioning-install-docker" {
  connection {
    user        = "${var.ssh_manager_user}"
    host        = "${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo yum install -q -y yum-utils",
      "sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo",
      "sudo yum makecache fast -q",
      "sudo yum install -q -y device-mapper-persistent-data lvm2",
      "sudo yum install -q -y docker-ce",
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

resource "null_resource" "janus-montoring-provisioning-config-docker" {
  depends_on = ["null_resource.janus-montoring-provisioning-install-docker"]
  count      = "${var.http_proxy != "" ? 1 : 0 }"

  connection {
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

resource "null_resource" "janus-montoring-provisioning-start-monitoring-statsd-grafana" {
  depends_on = ["null_resource.janus-montoring-provisioning-config-docker"]

  connection {
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
      "docker run -d -p 80:80 -p 8125:8125/udp -p 8126:8126 --name kamon-grafana-dashboard kamon/grafana_graphite | cat",

      "sleep 30",
      "set -x",
      "curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/datasources -H 'Content-type: application/json' -X POST -d '{\"Name\":\"graphite\",\"Type\":\"graphite\",\"IsDefault\":true,\"Url\":\"http://localhost:8000\",\"Access\":\"proxy\",\"BasicAuth\":false}'",
      "curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/datasources -H 'Content-type: application/json' -X POST -d '{\"Name\":\"prometheus\",\"Type\":\"prometheus\",\"IsDefault\":false,\"Url\":\"http://${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}:9090\",\"Access\":\"proxy\",\"BasicAuth\":false}'",
      "for g in $$(find /tmp/grafana_dashboards -type f -name '*.json'); do curl --noproxy 127.0.0.1 http://admin:admin@127.0.0.1/api/dashboards/db -H 'Content-type: application/json' -X POST -d @$${g}; done",
    ]
  }
}
resource "null_resource" "janus-montoring-provisioning-start-monitoring-prometheus" {
  depends_on = ["null_resource.janus-montoring-provisioning-config-docker"]

  connection {
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
      "docker run -d -p 9090:9090 -v /home/${var.ssh_manager_user}/prometheus.yml:/etc/prometheus/prometheus.yml --name prometheus prom/prometheus | cat",
    ]
  }
}

output "janus-monitoring" {
  value = "http://${openstack_compute_floatingip_associate_v2.janus-monitoring-fip.floating_ip}"
}
