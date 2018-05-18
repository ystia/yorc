#!/bin/bash -e
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#      http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


partition_number=1
volume_id=${VOLUME_ID}
partition_type=${PARTITION_TYPE}
device_name=${DEVICE}

echo $partition_type
echo $device_name

if [ -z "${volume_id}" ]; then
    echo "Creating disk partition on device ${device_name}"
    (echo n; echo p; echo ${partition_number}; echo ; echo ; echo t; echo ${partition_type}; echo w) | sudo fdisk ${device_name}
else
    echo "Not partitioning device since 'volume_id' is not empty"
fi

# Set this runtime property on the source (the filesystem)
# its needed by subsequent scripts
# ctx source instance runtime-properties filesys ${device_name}${partition_number}
export PARTITION_NAME=${device_name}${partition_number}

echo $PARTITION_NAME
