{
      "service": {
        "name": "yorc",
        "tags": ["${yorc_id}"],
        "address": "${yorc_ip}",
        "port": 8800,
        "check": {
          "name": "TCP check on port 8800",
          "tcp": "localhost:8800",
          "interval": "10s"
        }
      }
    }