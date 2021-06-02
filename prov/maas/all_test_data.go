// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package maas

var allocateResponse = `
{
    "hostname": "alert-buck",
    "distro_series": "",
    "address_ttl": null,
    "network_test_status": -1,
    "ip_addresses": [],
    "other_test_status_name": "Unknown",
    "memory_test_status_name": "Unknown",
    "cpu_count": 8,
    "hwe_kernel": null,
    "storage_test_status": 2,
    "node_type_name": "Machine",
    "commissioning_status": 2,
    "virtualmachine_id": null,
    "status_action": "",
    "bios_boot_method": "pxe",
    "netboot": true,
    "interface_set": [
        {
            "params": "",
            "interface_speed": 1000,
            "vendor": "Intel Corporation",
            "enabled": true,
            "id": 16,
            "sriov_max_vf": 7,
            "effective_mtu": 1500,
            "system_id": "tdgqkw",
            "type": "physical",
            "vlan": {
                "vid": 0,
                "mtu": 1500,
                "dhcp_on": true,
                "external_dhcp": null,
                "relay_vlan": null,
                "secondary_rack": null,
                "fabric": "fabric-1",
                "fabric_id": 1,
                "primary_rack": "67mdw3",
                "name": "untagged",
                "space": "undefined",
                "id": 5002,
                "resource_uri": "/MAAS/api/2.0/vlans/5002/"
            },
            "product": "82576 Gigabit Network Connection",
            "discovered": [],
            "links": [
                {
                    "id": 31,
                    "mode": "auto",
                    "subnet": {
                        "name": "172.20.0.0/24",
                        "description": "",
                        "vlan": {
                            "vid": 0,
                            "mtu": 1500,
                            "dhcp_on": true,
                            "external_dhcp": null,
                            "relay_vlan": null,
                            "secondary_rack": null,
                            "fabric": "fabric-1",
                            "fabric_id": 1,
                            "primary_rack": "67mdw3",
                            "name": "untagged",
                            "space": "undefined",
                            "id": 5002,
                            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                        },
                        "cidr": "172.20.0.0/24",
                        "rdns_mode": 2,
                        "gateway_ip": "172.20.0.1",
                        "dns_servers": [],
                        "allow_dns": true,
                        "allow_proxy": true,
                        "active_discovery": false,
                        "managed": true,
                        "space": "undefined",
                        "id": 2,
                        "resource_uri": "/MAAS/api/2.0/subnets/2/"
                    }
                }
            ],
            "numa_node": 0,
            "tags": [
                "sriov"
            ],
            "name": "enp1s0f0",
            "mac_address": "cb:d5:c9:f7:78:04",
            "firmware_version": "1.2.3",
            "parents": [],
            "children": [],
            "link_connected": true,
            "link_speed": 100,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
        },
        {
            "params": "",
            "interface_speed": 1000,
            "vendor": "Intel Corporation",
            "enabled": true,
            "id": 18,
            "sriov_max_vf": 7,
            "effective_mtu": 1500,
            "system_id": "tdgqkw",
            "type": "physical",
            "vlan": null,
            "product": "82576 Gigabit Network Connection",
            "discovered": null,
            "links": [],
            "numa_node": 0,
            "tags": [
                "sriov"
            ],
            "name": "enp1s0f1",
            "mac_address": "cb:d5:c9:f7:78:04",
            "firmware_version": "1.2.3",
            "parents": [],
            "children": [],
            "link_connected": true,
            "link_speed": 0,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/18/"
        }
    ],
    "network_test_status_name": "Unknown",
    "pod": null,
    "disable_ipv4": false,
    "blockdevice_set": [
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "storage_pool": null,
            "id": 11,
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "available_size": 0,
            "name": "sda",
            "uuid": null,
            "serial": "Z1W1QQ4T",
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "system_id": "tdgqkw",
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "type": "partition",
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "id": 7,
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "path": "/dev/disk/by-dname/sda",
            "used_size": 1000200994816,
            "partition_table_type": "GPT",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "storage_pool": null,
            "id": 12,
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "available_size": 1000204886016,
            "name": "sdb",
            "uuid": null,
            "serial": "Z1W1QJMW",
            "partitions": [],
            "path": "/dev/disk/by-dname/sdb",
            "used_size": 0,
            "partition_table_type": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "node_type": 0,
    "commissioning_status_name": "Passed",
    "interface_test_status": -1,
    "storage": 2000409.7720320001,
    "owner_data": {},
    "owner": "jim",
    "system_id": "tdgqkw",
    "raids": [],
    "min_hwe_kernel": "",
    "cpu_speed": 2930,
    "locked": false,
    "osystem": "",
    "physicalblockdevice_set": [
        {
            "firmware_version": "SN03",
            "storage_pool": null,
            "id": 11,
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "tags": [
                "ssd"
            ],
            "available_size": 0,
            "size": 1000204886016,
            "name": "sda",
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "uuid": null,
            "serial": "Z1W1QQ4T",
            "block_size": 512,
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "system_id": "tdgqkw",
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "type": "partition",
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "id": 7,
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "path": "/dev/disk/by-dname/sda",
            "used_size": 1000200994816,
            "partition_table_type": "GPT",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "firmware_version": "SN03",
            "storage_pool": null,
            "id": 12,
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "tags": [
                "ssd"
            ],
            "available_size": 1000204886016,
            "size": 1000204886016,
            "name": "sdb",
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "uuid": null,
            "serial": "Z1W1QJMW",
            "block_size": 512,
            "partitions": [],
            "path": "/dev/disk/by-dname/sdb",
            "used_size": 0,
            "partition_table_type": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "interface_test_status_name": "Unknown",
    "boot_interface": {
        "params": "",
        "interface_speed": 1000,
        "vendor": "Intel Corporation",
        "enabled": true,
        "id": 16,
        "sriov_max_vf": 7,
        "effective_mtu": 1500,
        "system_id": "tdgqkw",
        "type": "physical",
        "vlan": {
            "vid": 0,
            "mtu": 1500,
            "dhcp_on": true,
            "external_dhcp": null,
            "relay_vlan": null,
            "secondary_rack": null,
            "fabric": "fabric-1",
            "fabric_id": 1,
            "primary_rack": "67mdw3",
            "name": "untagged",
            "space": "undefined",
            "id": 5002,
            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
        },
        "product": "82576 Gigabit Network Connection",
        "discovered": [],
        "links": [
            {
                "id": 31,
                "mode": "auto",
                "subnet": {
                    "name": "172.20.0.0/24",
                    "description": "",
                    "vlan": {
                        "vid": 0,
                        "mtu": 1500,
                        "dhcp_on": true,
                        "external_dhcp": null,
                        "relay_vlan": null,
                        "secondary_rack": null,
                        "fabric": "fabric-1",
                        "fabric_id": 1,
                        "primary_rack": "67mdw3",
                        "name": "untagged",
                        "space": "undefined",
                        "id": 5002,
                        "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                    },
                    "cidr": "172.20.0.0/24",
                    "rdns_mode": 2,
                    "gateway_ip": "172.20.0.1",
                    "dns_servers": [],
                    "allow_dns": true,
                    "allow_proxy": true,
                    "active_discovery": false,
                    "managed": true,
                    "space": "undefined",
                    "id": 2,
                    "resource_uri": "/MAAS/api/2.0/subnets/2/"
                }
            }
        ],
        "numa_node": 0,
        "tags": [
            "sriov"
        ],
        "name": "enp1s0f0",
        "mac_address": "cb:d5:c9:f7:78:04",
        "firmware_version": "1.2.3",
        "parents": [],
        "children": [],
        "link_connected": true,
        "link_speed": 100,
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
    },
    "volume_groups": [],
    "storage_test_status_name": "Passed",
    "tag_names": [
        "yorc"
    ],
    "status_message": "Released",
    "hardware_info": {
        "system_vendor": "Bull SAS",
        "system_product": "bullx",
        "system_family": "Server",
        "system_version": "R422-E2",
        "system_sku": "R4222X80",
        "system_serial": "1234567890",
        "cpu_model": "Intel(R) Xeon(R) CPU           X5570 ",
        "mainboard_vendor": "Supermicro",
        "mainboard_product": "X8DTT",
        "mainboard_serial": "1234567890",
        "mainboard_version": "1.00",
        "mainboard_firmware_vendor": "American Megatrends Inc.",
        "mainboard_firmware_date": "05/20/2010",
        "mainboard_firmware_version": "R4222X80",
        "chassis_vendor": "Supermicro",
        "chassis_type": "Main Server Chassis",
        "chassis_serial": "1234567890.",
        "chassis_version": "1234567890"
    },
    "special_filesystems": [],
    "cpu_test_status_name": "Unknown",
    "current_commissioning_result_id": 6,
    "constraints_by_type": {
        "storage": {
            "label": [
                11
            ]
        }
    },
    "memory": 12288,
    "status": 10,
    "virtualblockdevice_set": [],
    "hardware_uuid": "1d030221-ce00-d553-d522-003048c6f252",
    "pool": {
        "name": "default",
        "description": "Default pool",
        "id": 0,
        "resource_uri": "/MAAS/api/2.0/resourcepool/0/"
    },
    "architecture": "amd64/generic",
    "iscsiblockdevice_set": [],
    "current_testing_result_id": 22,
    "memory_test_status": -1,
    "power_state": "off",
    "bcaches": [],
    "numanode_set": [
        {
            "index": 0,
            "memory": 7963,
            "cores": [
                0,
                1,
                2,
                3
            ],
            "hugepages_set": []
        },
        {
            "index": 1,
            "memory": 4028,
            "cores": [
                4,
                5,
                6,
                7
            ],
            "hugepages_set": []
        }
    ],
    "power_type": "ipmi",
    "fqdn": "alert-buck.maas",
    "cache_sets": [],
    "testing_status": 2,
    "current_installation_result_id": null,
    "default_gateways": {
        "ipv4": {
            "gateway_ip": "172.20.0.1",
            "link_id": null
        },
        "ipv6": {
            "gateway_ip": null,
            "link_id": null
        }
    },
    "zone": {
        "name": "default",
        "description": "",
        "id": 1,
        "resource_uri": "/MAAS/api/2.0/zones/default/"
    },
    "boot_disk": {
        "firmware_version": "SN03",
        "storage_pool": null,
        "id": 11,
        "system_id": "tdgqkw",
        "used_for": "GPT partitioned with 1 partition",
        "model": "ST1000NM0033-9ZM",
        "type": "physical",
        "filesystem": null,
        "numa_node": 0,
        "tags": [
            "ssd"
        ],
        "available_size": 0,
        "size": 1000204886016,
        "name": "sda",
        "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
        "uuid": null,
        "serial": "Z1W1QQ4T",
        "block_size": 512,
        "partitions": [
            {
                "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                "size": 1000194703360,
                "bootable": false,
                "tags": [],
                "system_id": "tdgqkw",
                "device_id": 11,
                "used_for": "ext4 formatted filesystem mounted at /",
                "type": "partition",
                "path": "/dev/disk/by-dname/sda-part2",
                "filesystem": {
                    "fstype": "ext4",
                    "label": "root",
                    "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                    "mount_point": "/",
                    "mount_options": null
                },
                "id": 7,
                "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
            }
        ],
        "path": "/dev/disk/by-dname/sda",
        "used_size": 1000200994816,
        "partition_table_type": "GPT",
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
    },
    "status_name": "Allocated",
    "domain": {
        "authoritative": true,
        "ttl": null,
        "is_default": true,
        "resource_record_count": 3,
        "name": "maas",
        "id": 0,
        "resource_uri": "/MAAS/api/2.0/domains/0/"
    },
    "description": "",
    "swap_size": null,
    "other_test_status": -1,
    "testing_status_name": "Passed",
    "cpu_test_status": -1,
    "resource_uri": "/MAAS/api/2.0/machines/tdgqkw/"
}
`

var deployResponse = ` 
{
    "physicalblockdevice_set": [
        {
            "firmware_version": "SN03",
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "path": "/dev/disk/by-dname/sda",
            "available_size": 0,
            "type": "physical",
            "tags": [
                "ssd"
            ],
            "partition_table_type": "GPT",
            "model": "ST1000NM0033-9ZM",
            "name": "sda",
            "filesystem": null,
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "used_size": 1000200994816,
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "system_id": "tdgqkw",
                    "type": "partition",
                    "id": 7,
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "serial": "Z1W1QQ4T",
            "block_size": 512,
            "uuid": null,
            "size": 1000204886016,
            "id": 11,
            "numa_node": 0,
            "storage_pool": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "firmware_version": "SN03",
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "path": "/dev/disk/by-dname/sdb",
            "available_size": 1000204886016,
            "type": "physical",
            "tags": [
                "ssd"
            ],
            "partition_table_type": null,
            "model": "ST1000NM0033-9ZM",
            "name": "sdb",
            "filesystem": null,
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "used_size": 0,
            "partitions": [],
            "serial": "Z1W1QJMW",
            "block_size": 512,
            "uuid": null,
            "size": 1000204886016,
            "id": 12,
            "numa_node": 0,
            "storage_pool": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "system_id": "tdgqkw",
    "other_test_status": -1,
    "power_type": "ipmi",
    "network_test_status_name": "Unknown",
    "memory_test_status_name": "Unknown",
    "network_test_status": -1,
    "boot_interface": {
        "link_connected": true,
        "system_id": "tdgqkw",
        "params": "",
        "links": [
            {
                "id": 31,
                "mode": "auto",
                "subnet": {
                    "name": "172.20.0.0/24",
                    "description": "",
                    "vlan": {
                        "vid": 0,
                        "mtu": 1500,
                        "dhcp_on": true,
                        "external_dhcp": null,
                        "relay_vlan": null,
                        "secondary_rack": null,
                        "primary_rack": "67mdw3",
                        "fabric": "fabric-1",
                        "id": 5002,
                        "space": "undefined",
                        "name": "untagged",
                        "fabric_id": 1,
                        "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                    },
                    "cidr": "172.20.0.0/24",
                    "rdns_mode": 2,
                    "gateway_ip": "172.20.0.1",
                    "dns_servers": [],
                    "allow_dns": true,
                    "allow_proxy": true,
                    "active_discovery": false,
                    "managed": true,
                    "id": 2,
                    "space": "undefined",
                    "resource_uri": "/MAAS/api/2.0/subnets/2/"
                }
            }
        ],
        "type": "physical",
        "mac_address": "cb:d5:c9:f7:78:04",
        "sriov_max_vf": 7,
        "tags": [
            "sriov"
        ],
        "children": [],
        "firmware_version": "1.2.3",
        "name": "enp1s0f0",
        "effective_mtu": 1500,
        "enabled": true,
        "link_speed": 100,
        "discovered": [],
        "vlan": {
            "vid": 0,
            "mtu": 1500,
            "dhcp_on": true,
            "external_dhcp": null,
            "relay_vlan": null,
            "secondary_rack": null,
            "primary_rack": "67mdw3",
            "fabric": "fabric-1",
            "id": 5002,
            "space": "undefined",
            "name": "untagged",
            "fabric_id": 1,
            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
        },
        "vendor": "Intel Corporation",
        "interface_speed": 1000,
        "id": 16,
        "parents": [],
        "numa_node": 0,
        "product": "82576 Gigabit Network Connection",
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
    },
    "address_ttl": null,
    "tag_names": [
        "yorc"
    ],
    "locked": false,
    "memory": 12288,
    "current_installation_result_id": 143,
    "domain": {
        "authoritative": true,
        "ttl": null,
        "is_default": true,
        "id": 0,
        "resource_record_count": 3,
        "name": "maas",
        "resource_uri": "/MAAS/api/2.0/domains/0/"
    },
    "virtualmachine_id": null,
    "description": "",
    "current_testing_result_id": 22,
    "storage_test_status_name": "Passed",
    "min_hwe_kernel": "",
    "bcaches": [],
    "memory_test_status": -1,
    "architecture": "amd64/generic",
    "status": 9,
    "pool": {
        "name": "default",
        "description": "Default pool",
        "id": 0,
        "resource_uri": "/MAAS/api/2.0/resourcepool/0/"
    },
    "iscsiblockdevice_set": [],
    "raids": [],
    "status_message": "Deploying",
    "swap_size": null,
    "testing_status": 2,
    "storage": 2000409.7720320001,
    "osystem": "ubuntu",
    "boot_disk": {
        "firmware_version": "SN03",
        "system_id": "tdgqkw",
        "used_for": "GPT partitioned with 1 partition",
        "path": "/dev/disk/by-dname/sda",
        "available_size": 0,
        "type": "physical",
        "tags": [
            "ssd"
        ],
        "partition_table_type": "GPT",
        "model": "ST1000NM0033-9ZM",
        "name": "sda",
        "filesystem": null,
        "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
        "used_size": 1000200994816,
        "partitions": [
            {
                "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                "size": 1000194703360,
                "bootable": false,
                "tags": [],
                "path": "/dev/disk/by-dname/sda-part2",
                "filesystem": {
                    "fstype": "ext4",
                    "label": "root",
                    "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                    "mount_point": "/",
                    "mount_options": null
                },
                "system_id": "tdgqkw",
                "type": "partition",
                "id": 7,
                "device_id": 11,
                "used_for": "ext4 formatted filesystem mounted at /",
                "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
            }
        ],
        "serial": "Z1W1QQ4T",
        "block_size": 512,
        "uuid": null,
        "size": 1000204886016,
        "id": 11,
        "numa_node": 0,
        "storage_pool": null,
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
    },
    "power_state": "off",
    "special_filesystems": [],
    "volume_groups": [],
    "commissioning_status_name": "Passed",
    "hwe_kernel": "ga-20.04",
    "netboot": true,
    "blockdevice_set": [
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "path": "/dev/disk/by-dname/sda",
            "available_size": 0,
            "type": "physical",
            "partition_table_type": "GPT",
            "model": "ST1000NM0033-9ZM",
            "name": "sda",
            "filesystem": null,
            "used_size": 1000200994816,
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "system_id": "tdgqkw",
                    "type": "partition",
                    "id": 7,
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "serial": "Z1W1QQ4T",
            "uuid": null,
            "id": 11,
            "numa_node": 0,
            "storage_pool": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "path": "/dev/disk/by-dname/sdb",
            "available_size": 1000204886016,
            "type": "physical",
            "partition_table_type": null,
            "model": "ST1000NM0033-9ZM",
            "name": "sdb",
            "filesystem": null,
            "used_size": 0,
            "partitions": [],
            "serial": "Z1W1QJMW",
            "uuid": null,
            "id": 12,
            "numa_node": 0,
            "storage_pool": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "cpu_count": 8,
    "pod": null,
    "ip_addresses": [],
    "node_type": 0,
    "distro_series": "focal",
    "current_commissioning_result_id": 6,
    "fqdn": "alert-buck.maas",
    "owner_data": {},
    "virtualblockdevice_set": [],
    "default_gateways": {
        "ipv4": {
            "gateway_ip": "172.20.0.1",
            "link_id": null
        },
        "ipv6": {
            "gateway_ip": null,
            "link_id": null
        }
    },
    "cpu_test_status_name": "Unknown",
    "hostname": "alert-buck",
    "interface_test_status": -1,
    "cpu_speed": 2930,
    "status_action": "",
    "hardware_info": {
        "system_vendor": "Bull SAS",
        "system_product": "bullx",
        "system_family": "Server",
        "system_version": "R422-E2",
        "system_sku": "R4222X80",
        "system_serial": "1234567890",
        "cpu_model": "Intel(R) Xeon(R) CPU           X5570 ",
        "mainboard_vendor": "Supermicro",
        "mainboard_product": "X8DTT",
        "mainboard_serial": "1234567890",
        "mainboard_version": "1.00",
        "mainboard_firmware_vendor": "American Megatrends Inc.",
        "mainboard_firmware_date": "05/20/2010",
        "mainboard_firmware_version": "R4222X80",
        "chassis_vendor": "Supermicro",
        "chassis_type": "Main Server Chassis",
        "chassis_serial": "1234567890.",
        "chassis_version": "1234567890"
    },
    "owner": "jim",
    "interface_set": [
        {
            "link_connected": true,
            "system_id": "tdgqkw",
            "params": "",
            "links": [
                {
                    "id": 31,
                    "mode": "auto",
                    "subnet": {
                        "name": "172.20.0.0/24",
                        "description": "",
                        "vlan": {
                            "vid": 0,
                            "mtu": 1500,
                            "dhcp_on": true,
                            "external_dhcp": null,
                            "relay_vlan": null,
                            "secondary_rack": null,
                            "primary_rack": "67mdw3",
                            "fabric": "fabric-1",
                            "id": 5002,
                            "space": "undefined",
                            "name": "untagged",
                            "fabric_id": 1,
                            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                        },
                        "cidr": "172.20.0.0/24",
                        "rdns_mode": 2,
                        "gateway_ip": "172.20.0.1",
                        "dns_servers": [],
                        "allow_dns": true,
                        "allow_proxy": true,
                        "active_discovery": false,
                        "managed": true,
                        "id": 2,
                        "space": "undefined",
                        "resource_uri": "/MAAS/api/2.0/subnets/2/"
                    }
                }
            ],
            "type": "physical",
            "mac_address": "cb:d5:c9:f7:78:04",
            "sriov_max_vf": 7,
            "tags": [
                "sriov"
            ],
            "children": [],
            "firmware_version": "1.2.3",
            "name": "enp1s0f0",
            "effective_mtu": 1500,
            "enabled": true,
            "link_speed": 100,
            "discovered": [],
            "vlan": {
                "vid": 0,
                "mtu": 1500,
                "dhcp_on": true,
                "external_dhcp": null,
                "relay_vlan": null,
                "secondary_rack": null,
                "primary_rack": "67mdw3",
                "fabric": "fabric-1",
                "id": 5002,
                "space": "undefined",
                "name": "untagged",
                "fabric_id": 1,
                "resource_uri": "/MAAS/api/2.0/vlans/5002/"
            },
            "vendor": "Intel Corporation",
            "interface_speed": 1000,
            "id": 16,
            "parents": [],
            "numa_node": 0,
            "product": "82576 Gigabit Network Connection",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
        },
        {
            "link_connected": true,
            "system_id": "tdgqkw",
            "params": "",
            "links": [],
            "type": "physical",
            "mac_address": "cb:d5:c9:f7:78:04",
            "sriov_max_vf": 7,
            "tags": [
                "sriov"
            ],
            "children": [],
            "firmware_version": "1.2.3",
            "name": "enp1s0f1",
            "effective_mtu": 1500,
            "enabled": true,
            "link_speed": 0,
            "discovered": null,
            "vlan": null,
            "vendor": "Intel Corporation",
            "interface_speed": 1000,
            "id": 18,
            "parents": [],
            "numa_node": 0,
            "product": "82576 Gigabit Network Connection",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/18/"
        }
    ],
    "commissioning_status": 2,
    "zone": {
        "name": "default",
        "description": "",
        "id": 1,
        "resource_uri": "/MAAS/api/2.0/zones/default/"
    },
    "node_type_name": "Machine",
    "interface_test_status_name": "Unknown",
    "bios_boot_method": "pxe",
    "status_name": "Deploying",
    "cache_sets": [],
    "disable_ipv4": false,
    "storage_test_status": 2,
    "hardware_uuid": "1d030221-ce00-d553-d522-003048c6f252",
    "testing_status_name": "Passed",
    "cpu_test_status": -1,
    "numanode_set": [
        {
            "index": 0,
            "memory": 7963,
            "cores": [
                0,
                1,
                2,
                3
            ],
            "hugepages_set": []
        },
        {
            "index": 1,
            "memory": 4028,
            "cores": [
                4,
                5,
                6,
                7
            ],
            "hugepages_set": []
        }
    ],
    "other_test_status_name": "Unknown",
    "resource_uri": "/MAAS/api/2.0/machines/tdgqkw/"
}
`

var machineDeployed = ` 
{
    "hostname": "alert-buck",
    "distro_series": "focal",
    "address_ttl": null,
    "network_test_status": -1,
    "ip_addresses": [
        "172.20.0.114"
    ],
    "other_test_status_name": "Unknown",
    "memory_test_status_name": "Unknown",
    "cpu_count": 8,
    "hwe_kernel": "ga-20.04",
    "storage_test_status": 2,
    "node_type_name": "Machine",
    "commissioning_status": 2,
    "virtualmachine_id": null,
    "status_action": "",
    "bios_boot_method": "pxe",
    "netboot": false,
    "interface_set": [
        {
            "params": "",
            "interface_speed": 1000,
            "vendor": "Intel Corporation",
            "enabled": true,
            "id": 16,
            "sriov_max_vf": 7,
            "effective_mtu": 1500,
            "system_id": "tdgqkw",
            "type": "physical",
            "vlan": {
                "vid": 0,
                "mtu": 1500,
                "dhcp_on": true,
                "external_dhcp": null,
                "relay_vlan": null,
                "secondary_rack": null,
                "fabric": "fabric-1",
                "fabric_id": 1,
                "primary_rack": "67mdw3",
                "name": "untagged",
                "space": "undefined",
                "id": 5002,
                "resource_uri": "/MAAS/api/2.0/vlans/5002/"
            },
            "product": "82576 Gigabit Network Connection",
            "discovered": [],
            "links": [
                {
                    "id": 31,
                    "mode": "auto",
                    "ip_address": "172.20.0.114",
                    "subnet": {
                        "name": "172.20.0.0/24",
                        "description": "",
                        "vlan": {
                            "vid": 0,
                            "mtu": 1500,
                            "dhcp_on": true,
                            "external_dhcp": null,
                            "relay_vlan": null,
                            "secondary_rack": null,
                            "fabric": "fabric-1",
                            "fabric_id": 1,
                            "primary_rack": "67mdw3",
                            "name": "untagged",
                            "space": "undefined",
                            "id": 5002,
                            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                        },
                        "cidr": "172.20.0.0/24",
                        "rdns_mode": 2,
                        "gateway_ip": "172.20.0.1",
                        "dns_servers": [],
                        "allow_dns": true,
                        "allow_proxy": true,
                        "active_discovery": false,
                        "managed": true,
                        "space": "undefined",
                        "id": 2,
                        "resource_uri": "/MAAS/api/2.0/subnets/2/"
                    }
                }
            ],
            "numa_node": 0,
            "tags": [
                "sriov"
            ],
            "name": "enp1s0f0",
            "mac_address": "cb:d5:c9:f7:78:04",
            "firmware_version": "1.2.3",
            "parents": [],
            "children": [],
            "link_connected": true,
            "link_speed": 100,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
        },
        {
            "params": "",
            "interface_speed": 1000,
            "vendor": "Intel Corporation",
            "enabled": true,
            "id": 18,
            "sriov_max_vf": 7,
            "effective_mtu": 1500,
            "system_id": "tdgqkw",
            "type": "physical",
            "vlan": null,
            "product": "82576 Gigabit Network Connection",
            "discovered": null,
            "links": [],
            "numa_node": 0,
            "tags": [
                "sriov"
            ],
            "name": "enp1s0f1",
            "mac_address": "cb:d5:c9:f7:78:04",
            "firmware_version": "1.2.3",
            "parents": [],
            "children": [],
            "link_connected": true,
            "link_speed": 0,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/18/"
        }
    ],
    "network_test_status_name": "Unknown",
    "pod": null,
    "disable_ipv4": false,
    "blockdevice_set": [
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "storage_pool": null,
            "id": 11,
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "available_size": 0,
            "name": "sda",
            "uuid": null,
            "serial": "Z1W1QQ4T",
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "system_id": "tdgqkw",
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "type": "partition",
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "id": 7,
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "path": "/dev/disk/by-dname/sda",
            "used_size": 1000200994816,
            "partition_table_type": "GPT",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "size": 1000204886016,
            "block_size": 512,
            "tags": [
                "ssd"
            ],
            "storage_pool": null,
            "id": 12,
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "available_size": 1000204886016,
            "name": "sdb",
            "uuid": null,
            "serial": "Z1W1QJMW",
            "partitions": [],
            "path": "/dev/disk/by-dname/sdb",
            "used_size": 0,
            "partition_table_type": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "node_type": 0,
    "commissioning_status_name": "Passed",
    "interface_test_status": -1,
    "storage": 2000409.7720320001,
    "owner_data": {},
    "owner": "jim",
    "system_id": "tdgqkw",
    "raids": [],
    "min_hwe_kernel": "",
    "cpu_speed": 2930,
    "locked": false,
    "osystem": "ubuntu",
    "physicalblockdevice_set": [
        {
            "firmware_version": "SN03",
            "storage_pool": null,
            "id": 11,
            "system_id": "tdgqkw",
            "used_for": "GPT partitioned with 1 partition",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "tags": [
                "ssd"
            ],
            "available_size": 0,
            "size": 1000204886016,
            "name": "sda",
            "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
            "uuid": null,
            "serial": "Z1W1QQ4T",
            "block_size": 512,
            "partitions": [
                {
                    "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                    "size": 1000194703360,
                    "bootable": false,
                    "tags": [],
                    "system_id": "tdgqkw",
                    "device_id": 11,
                    "used_for": "ext4 formatted filesystem mounted at /",
                    "type": "partition",
                    "path": "/dev/disk/by-dname/sda-part2",
                    "filesystem": {
                        "fstype": "ext4",
                        "label": "root",
                        "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                        "mount_point": "/",
                        "mount_options": null
                    },
                    "id": 7,
                    "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
                }
            ],
            "path": "/dev/disk/by-dname/sda",
            "used_size": 1000200994816,
            "partition_table_type": "GPT",
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
        },
        {
            "firmware_version": "SN03",
            "storage_pool": null,
            "id": 12,
            "system_id": "tdgqkw",
            "used_for": "Unused",
            "model": "ST1000NM0033-9ZM",
            "type": "physical",
            "filesystem": null,
            "numa_node": 0,
            "tags": [
                "ssd"
            ],
            "available_size": 1000204886016,
            "size": 1000204886016,
            "name": "sdb",
            "id_path": "/dev/disk/by-id/wwn-0x5000c5006689ae3f",
            "uuid": null,
            "serial": "Z1W1QJMW",
            "block_size": 512,
            "partitions": [],
            "path": "/dev/disk/by-dname/sdb",
            "used_size": 0,
            "partition_table_type": null,
            "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/12/"
        }
    ],
    "interface_test_status_name": "Unknown",
    "boot_interface": {
        "params": "",
        "interface_speed": 1000,
        "vendor": "Intel Corporation",
        "enabled": true,
        "id": 16,
        "sriov_max_vf": 7,
        "effective_mtu": 1500,
        "system_id": "tdgqkw",
        "type": "physical",
        "vlan": {
            "vid": 0,
            "mtu": 1500,
            "dhcp_on": true,
            "external_dhcp": null,
            "relay_vlan": null,
            "secondary_rack": null,
            "fabric": "fabric-1",
            "fabric_id": 1,
            "primary_rack": "67mdw3",
            "name": "untagged",
            "space": "undefined",
            "id": 5002,
            "resource_uri": "/MAAS/api/2.0/vlans/5002/"
        },
        "product": "82576 Gigabit Network Connection",
        "discovered": [],
        "links": [
            {
                "id": 31,
                "mode": "auto",
                "ip_address": "172.20.0.114",
                "subnet": {
                    "name": "172.20.0.0/24",
                    "description": "",
                    "vlan": {
                        "vid": 0,
                        "mtu": 1500,
                        "dhcp_on": true,
                        "external_dhcp": null,
                        "relay_vlan": null,
                        "secondary_rack": null,
                        "fabric": "fabric-1",
                        "fabric_id": 1,
                        "primary_rack": "67mdw3",
                        "name": "untagged",
                        "space": "undefined",
                        "id": 5002,
                        "resource_uri": "/MAAS/api/2.0/vlans/5002/"
                    },
                    "cidr": "172.20.0.0/24",
                    "rdns_mode": 2,
                    "gateway_ip": "172.20.0.1",
                    "dns_servers": [],
                    "allow_dns": true,
                    "allow_proxy": true,
                    "active_discovery": false,
                    "managed": true,
                    "space": "undefined",
                    "id": 2,
                    "resource_uri": "/MAAS/api/2.0/subnets/2/"
                }
            }
        ],
        "numa_node": 0,
        "tags": [
            "sriov"
        ],
        "name": "enp1s0f0",
        "mac_address": "cb:d5:c9:f7:78:04",
        "firmware_version": "1.2.3",
        "parents": [],
        "children": [],
        "link_connected": true,
        "link_speed": 100,
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/interfaces/16/"
    },
    "volume_groups": [],
    "storage_test_status_name": "Passed",
    "tag_names": [
        "yorc"
    ],
    "status_message": "Deployed",
    "hardware_info": {
        "system_vendor": "Bull SAS",
        "system_product": "bullx",
        "system_family": "Server",
        "system_version": "R422-E2",
        "system_sku": "R4222X80",
        "system_serial": "1234567890",
        "cpu_model": "Intel(R) Xeon(R) CPU           X5570 ",
        "mainboard_vendor": "Supermicro",
        "mainboard_product": "X8DTT",
        "mainboard_serial": "1234567890",
        "mainboard_version": "1.00",
        "mainboard_firmware_vendor": "American Megatrends Inc.",
        "mainboard_firmware_date": "05/20/2010",
        "mainboard_firmware_version": "R4222X80",
        "chassis_vendor": "Supermicro",
        "chassis_type": "Main Server Chassis",
        "chassis_serial": "1234567890.",
        "chassis_version": "1234567890"
    },
    "special_filesystems": [],
    "cpu_test_status_name": "Unknown",
    "current_commissioning_result_id": 6,
    "memory": 12288,
    "status": 6,
    "virtualblockdevice_set": [],
    "hardware_uuid": "1d030221-ce00-d553-d522-003048c6f252",
    "pool": {
        "name": "default",
        "description": "Default pool",
        "id": 0,
        "resource_uri": "/MAAS/api/2.0/resourcepool/0/"
    },
    "architecture": "amd64/generic",
    "iscsiblockdevice_set": [],
    "current_testing_result_id": 22,
    "memory_test_status": -1,
    "power_state": "on",
    "bcaches": [],
    "numanode_set": [
        {
            "index": 0,
            "memory": 7963,
            "cores": [
                0,
                1,
                2,
                3
            ],
            "hugepages_set": []
        },
        {
            "index": 1,
            "memory": 4028,
            "cores": [
                4,
                5,
                6,
                7
            ],
            "hugepages_set": []
        }
    ],
    "power_type": "ipmi",
    "fqdn": "alert-buck.maas",
    "cache_sets": [],
    "testing_status": 2,
    "current_installation_result_id": 143,
    "default_gateways": {
        "ipv4": {
            "gateway_ip": "172.20.0.1",
            "link_id": null
        },
        "ipv6": {
            "gateway_ip": null,
            "link_id": null
        }
    },
    "zone": {
        "name": "default",
        "description": "",
        "id": 1,
        "resource_uri": "/MAAS/api/2.0/zones/default/"
    },
    "boot_disk": {
        "firmware_version": "SN03",
        "storage_pool": null,
        "id": 11,
        "system_id": "tdgqkw",
        "used_for": "GPT partitioned with 1 partition",
        "model": "ST1000NM0033-9ZM",
        "type": "physical",
        "filesystem": null,
        "numa_node": 0,
        "tags": [
            "ssd"
        ],
        "available_size": 0,
        "size": 1000204886016,
        "name": "sda",
        "id_path": "/dev/disk/by-id/wwn-0x5000c50066898892",
        "uuid": null,
        "serial": "Z1W1QQ4T",
        "block_size": 512,
        "partitions": [
            {
                "uuid": "32fc769e-2f84-4db6-8d06-3c0b6ef4282a",
                "size": 1000194703360,
                "bootable": false,
                "tags": [],
                "system_id": "tdgqkw",
                "device_id": 11,
                "used_for": "ext4 formatted filesystem mounted at /",
                "type": "partition",
                "path": "/dev/disk/by-dname/sda-part2",
                "filesystem": {
                    "fstype": "ext4",
                    "label": "root",
                    "uuid": "bc9577af-b6a9-4eb7-95bf-887300365bc9",
                    "mount_point": "/",
                    "mount_options": null
                },
                "id": 7,
                "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/partition/7"
            }
        ],
        "path": "/dev/disk/by-dname/sda",
        "used_size": 1000200994816,
        "partition_table_type": "GPT",
        "resource_uri": "/MAAS/api/2.0/nodes/tdgqkw/blockdevices/11/"
    },
    "status_name": "Deployed",
    "domain": {
        "authoritative": true,
        "ttl": null,
        "is_default": true,
        "resource_record_count": 3,
        "name": "maas",
        "id": 0,
        "resource_uri": "/MAAS/api/2.0/domains/0/"
    },
    "description": "",
    "swap_size": null,
    "other_test_status": -1,
    "testing_status_name": "Passed",
    "cpu_test_status": -1,
    "resource_uri": "/MAAS/api/2.0/machines/tdgqkw/"
}
`
