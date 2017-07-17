{
      "service": {
        "name": "janus",
        "tags": ["${janus_id}"],
        "address": "${janus_ip}",
        "port": 8800,
        "check": {
          "name": "TCP check on port 8800",
          "tcp": "localhost:8800",
          "interval": "10s"
        }
      }
    }