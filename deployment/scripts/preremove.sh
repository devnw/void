#!/bin/sh
set -e

stop_services() {
    systemctl stop void.service
}

disable_services() {
    systemctl disable void.service

    systemctl daemon-reload
}

case $1 in
    remove)
        stop_services
        disable_services
        ;;
    upgrade)
        stop_services
        ;;
    deconfigure)
        stop_services
        ;;
    failed-upgrade)
        ;;
esac
