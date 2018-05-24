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


volume_id=${VOLUME_ID}
fs_type=${FS_TYPE}
filesys=${PARTITION_NAME}

(eval $(blkid $DEV | awk ' { print $3 } '); echo $TYPE)

if [ -z "${volume_id}" ]; then
    mkfs_executable='mkfs.xfs'

    echo "Creating xfs file system using ${mkfs_executable}"
    sudo ${mkfs_executable} ${filesys}
else
    echo "Not making a filesystem since 'volume_id' not empty"
fi
