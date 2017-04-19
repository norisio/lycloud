#!/bin/sh

case "$1" in
	start)
		cd `dirname "${0}"`
		go build
		./lycloud &
		;;
	stop)
		cd `dirname "${0}"`
		killall lycloud
		;;
esac

exit 0
		
