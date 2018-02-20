[Unit]
Description=Yorc Server
After=consul.service autofs.service
Wants=consul.service autofs.service

[Service]
ExecStart=/usr/local/bin/yorc server
ExecReload=/bin/kill -s HUP $MAINPID
User=${user}
WorkingDirectory=/home/${user}


# restart the consul process if it exits prematurely
Restart=on-failure
StartLimitBurst=3
StartLimitInterval=60s

[Install]
WantedBy=multi-user.target

