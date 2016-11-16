#!/bin/bash -e

partition_number=1
volume_id=${VOLUME_ID}
partition_type=${PARTITION_TYPE}
device_name=${DEVICE}

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
