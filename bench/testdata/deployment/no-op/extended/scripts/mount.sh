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


fs_mount_path=${FS_MOUNT_PATH}
filesys=${PARTITION_NAME}

if [ ! -f ${fs_mount_path} ]; then
    sudo mkdir -p ${fs_mount_path}
fi

echo "Mounting file system ${filesys} on ${fs_mount_path}"
sudo mount ${filesys} ${fs_mount_path}

user=$(whoami)
echo "Changing ownership of ${fs_mount_path} to ${user}"
sudo chown ${user} ${fs_mount_path}
