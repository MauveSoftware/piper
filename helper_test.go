package main

import (
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestIsRouteDel(t *testing.T) {
	tests := []struct {
		name     string
		update   *netlink.RouteUpdate
		expected bool
	}{
		{
			name:     "delete update",
			update:   &netlink.RouteUpdate{Type: unix.RTM_DELROUTE},
			expected: true,
		},
		{
			name:     "add update",
			update:   &netlink.RouteUpdate{Type: unix.RTM_NEWROUTE},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRouteDel(tt.update)
			if got != tt.expected {
				t.Errorf("isRouteDel() = %v, want %v for update type %d", got, tt.expected, tt.update.Type)
			}
		})
	}
}

func TestRouteUpdateType(t *testing.T) {
	tests := []struct {
		name     string
		update   *netlink.RouteUpdate
		expected string
	}{
		{
			name:     "delete update maps to delete",
			update:   &netlink.RouteUpdate{Type: unix.RTM_DELROUTE},
			expected: "delete",
		},
		{
			name:     "new route update maps to add",
			update:   &netlink.RouteUpdate{Type: unix.RTM_NEWROUTE},
			expected: "add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routeUpdateType(tt.update)
			if got != tt.expected {
				t.Errorf("routeUpdateType() = %q, want %q for update type %d", got, tt.expected, tt.update.Type)
			}
		})
	}
}
