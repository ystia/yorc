{
      "service": {
        "name": "alien",
        "tags": ["alien4cloud"],
        "address": "${ip_address}",
        "port": 8088,
        "check": {
          "name": "TCP check on port 8088",
          "tcp": "localhost:8088",
          "interval": "30s"
        }
      }
    }