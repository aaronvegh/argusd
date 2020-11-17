#!/bin/sh

accessKey=$1
sudo sed -i ':a;N;$!ba;s/'"${accessKey}"'\n//g' /etc/argusd.conf

chars=$(wc -c < /etc/argusd.conf)
if [ $chars -eq 1 ]
then
	rm /etc/argusd.conf
fi

systemctl stop argusd.service
systemctl disable argusd.service

rm /usr/local/bin/argusd
rm /lib/systemd/system/argusd.service