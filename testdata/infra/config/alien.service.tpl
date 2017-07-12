[Unit]
Description=Alien4Cloud
After=network-online.target firewalld.service
Wants=network-online.target

[Service]
ExecStart=/home/${user}/alien4cloud/alien4cloud.sh
User=${user}
WorkingDirectory=/home/${user}/alien4cloud


# restart the consul process if it exits prematurely
Restart=on-failure
StartLimitBurst=3
StartLimitInterval=60s

[Install]
WantedBy=multi-user.target

