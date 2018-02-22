{
      "service": {
        "name": "nfs",
        "tags": ["nfs-server"],
        "address": "${ip_address}",
        "check": {
          "name": "NFS health check",
          "script": "systemctl status nfs-server.service && rpcinfo -p | grep 'mountd' && showmount -e | grep '/mountedStorageNFS/yorc-server/work'",
          "interval": "30s"
        }
      }
    }