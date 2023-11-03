#!/bin/sh
set -e

configure()
{
	# Check if /etc/systemd/resolved.conf exists before attempting to modify it
	if [ -f /etc/systemd/resolved.conf ]; then
		# Set DNSStubListener=no in /etc/systemd/resolved.conf
		sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf
	fi

	# Check if the systemd-resolved service exists and is active before attempting to restart it
	if systemctl is-active --quiet systemd-resolved; then
		# Restart the systemd-resolved service
		systemctl restart systemd-resolved
	fi
	
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
