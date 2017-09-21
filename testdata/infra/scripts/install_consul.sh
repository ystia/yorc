#!/usr/bin/env bash
set -ex

sudo yum install -q -y wget zip unzip
cd /tmp && wget -q https://releases.hashicorp.com/consul/0.8.4/consul_0.8.4_linux_amd64.zip && sudo unzip /tmp/consul_0.8.4_linux_amd64.zip -d /usr/local/bin
sudo mv /tmp/consul.service /etc/systemd/system/consul.service
sudo chown root:root /etc/systemd/system/consul.service
sudo mkdir -p /etc/consul.d
