#!/bin/bash -e

volume_id=${VOLUME_ID}
filesys=${PARTITION_NAME}

(eval $(sudo blkid ${filesys} | awk ' { print $3 } '); echo $TYPE)

if [ -z "${volume_id}" ]; then

    echo "Creating xfs file system using ${mkfs_executable}"
    sudo mkfs.xfs -f ${filesys}
else
    echo "Not making a filesystem since 'volume_id' not empty"
fi
