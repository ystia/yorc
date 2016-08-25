#!/bin/bash -e

fs_mount_path=${FS_MOUNT_PATH}

echo "Unmounting file system on ${fs_mount_path}"
sudo umount -l ${fs_mount_path}

echo "Removing ${fs_mount_path} directory"
sudo rmdir ${fs_mount_path}
