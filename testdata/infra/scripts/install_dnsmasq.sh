#!/usr/bin/env bash
set -ex

sudo yum install -q -y dnsmasq

sudo mv /etc/dnsmasq.conf /etc/dnsmasq.conf.ori
cat << EOF  | sudo tee /etc/dnsmasq.conf > /dev/null 
# Configuration file for dnsmasq.
#
# Format is one option per line, legal options are the same
# as the long options legal on the command line. See
# "/usr/sbin/dnsmasq --help" or "man 8 dnsmasq" for details.

# This file was automatically generated 

# If you don't want dnsmasq to read /etc/resolv.conf or any other
# file, getting its servers from this file instead (see below), then
# uncomment this.
#no-resolv

# Add other name servers here, with domain specs if they are for
# non-public domains.
server=/consul/127.0.0.1#8600
server=/arpa/127.0.0.1#8600

# If you don't want dnsmasq to read /etc/hosts, uncomment the
# following line.
no-hosts
        
resolv-file=/etc/resolv.dnsmasq.conf
EOF

sudo cp /etc/resolv.conf /etc/resolv.dnsmasq.conf
cat << EOF  | sudo tee /etc/resolv.conf > /dev/null 
# Automatically generated to enable Consul DNS resolution through dnsmasq
nameserver 127.0.0.1
search node.consul service.consul
EOF

sudo sed -i -e "/PEERDNS/ s/^/#/g" /etc/sysconfig/network-scripts/ifcfg-eth*
echo -e "\nPEERDNS=no\n" | sudo tee -a /etc/sysconfig/network-scripts/ifcfg-eth* > /dev/null 2>&1
# Enable and restart dnsmasq
sudo systemctl enable dnsmasq
sudo systemctl restart dnsmasq
