#!/bin/sh
set -e

case "$1" in
	start)
		cd `dirname "${0}"`
    rm -rf working/
		go build
		./lycloud &
		;;
	stop)
		cd `dirname "${0}"`
		killall lycloud
		;;
esac

exit 0
		
