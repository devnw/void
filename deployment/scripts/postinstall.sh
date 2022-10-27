#!/bin/sh
set -e

configure()
{
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
