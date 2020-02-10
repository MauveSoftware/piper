package main

import (
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func isRouteDel(u *netlink.RouteUpdate) bool {
	return u.Type == unix.RTM_DELROUTE
}

func routeUpdateType(u *netlink.RouteUpdate) string {
	if isRouteDel(u) {
		return "delete"
	}

	return "add"
}
