#!/bin/bash

defOS="linux"
defArch="amd64"

ACCESSKEY=$1
OS=${2:-$defOS}
ARCH=${3:-$defArch}

SCRIPTNAME="argusd-$OS-$ARCH"

echo -e "$ACCESSKEY\n" >> /etc/argusd.conf
chmod 600 /etc/argusd.conf

(cd /usr/local/bin/ && curl -O http://argusd.s3.amazonaws.com/argusd/$SCRIPTNAME)
mv /usr/local/bin/$SCRIPTNAME /usr/local/bin/argusd
chmod +x /usr/local/bin/argusd

servicefile="[Unit]
Description=argusd

[Service]
Type=simple
Restart=always
RestartSec=5s
ExecStart=/usr/local/bin/argusd

[Install]
WantedBy=multi-user.target"

printf '%s\n' $servicefile > /lib/systemd/system/argusd.service

systemctl start argusd.service
systemctl enable argusd.service