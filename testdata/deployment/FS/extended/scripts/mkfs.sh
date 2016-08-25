#!/bin/bash -e

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
