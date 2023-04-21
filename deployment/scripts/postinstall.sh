#!/bin/sh
set -e

configure()
{
	# Set DNSStubListener=no in /etc/systemd/resolved.conf
	sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf

	# Restart the systemd-resolved service
	systemctl restart systemd-resolved

	systemctl enable void.service
	
	systemctl daemon-reload
	
	systemctl start void.service
}

case $1 in
	configure)
		configure
		;;

	abort-upgrade)
		;;

	abort-remove)
		;;

	abort-deconfigure)
		;;
esac
