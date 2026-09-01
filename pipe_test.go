package main

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Note: tests here deliberately avoid any code path that reaches
// pipe.replaceRoute(), since that calls netlink.RouteReplace() against the
// real kernel routing table. Running that in a unit test would mutate system
// state (and requires CAP_NET_ADMIN), so it is out of scope for these tests.

func mustParseCIDR(t *testing.T, s string) net.IPNet {
	t.Helper()

	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("could not parse CIDR %q: %v", s, err)
	}

	return *n
}

func TestNewPipe(t *testing.T) {
	t.Run("regular prefix is kept", func(t *testing.T) {
		pfx := mustParseCIDR(t, "10.0.0.0/8")
		p := newPipe("pipe1", pfx, 254, 100, netlink.RouteProtocol(100))

		if p.prefix == nil {
			t.Fatal("prefix = nil, want non-nil for 10.0.0.0/8")
		}

		if p.prefix.String() != "10.0.0.0/8" {
			t.Errorf("prefix = %v, want 10.0.0.0/8", p.prefix)
		}

		if p.name != "pipe1" || p.sourceTable != 254 || p.targetTable != 100 {
			t.Errorf("unexpected pipe fields: %+v", p)
		}
	})

	t.Run("default route becomes nil prefix", func(t *testing.T) {
		pfx := mustParseCIDR(t, "0.0.0.0/0")
		p := newPipe("pipe1", pfx, 254, 100, netlink.RouteProtocol(100))

		if p.prefix != nil {
			t.Errorf("prefix = %v, want nil for default route", p.prefix)
		}
	})
}

func TestPipePrefixMatches(t *testing.T) {
	pfxA := mustParseCIDR(t, "10.0.0.0/8")
	pfxB := mustParseCIDR(t, "192.168.0.0/16")

	tests := []struct {
		name       string
		pipePrefix *net.IPNet
		update     *net.IPNet
		expected   bool
	}{
		{
			name:       "both default routes match",
			pipePrefix: nil,
			update:     nil,
			expected:   true,
		},
		{
			name:       "pipe default, update specific does not match",
			pipePrefix: nil,
			update:     &pfxA,
			expected:   false,
		},
		{
			name:       "pipe specific, update default does not match",
			pipePrefix: &pfxA,
			update:     nil,
			expected:   false,
		},
		{
			name:       "same prefix matches",
			pipePrefix: &pfxA,
			update:     &pfxA,
			expected:   true,
		},
		{
			name:       "different prefixes do not match",
			pipePrefix: &pfxA,
			update:     &pfxB,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &pipe{prefix: tt.pipePrefix}

			got := p.pefixMatches(tt.update)
			if got != tt.expected {
				t.Errorf("pefixMatches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPipeRouteEqual(t *testing.T) {
	base := netlink.Route{
		Gw:        net.ParseIP("10.0.0.1"),
		Src:       net.ParseIP("10.0.0.2"),
		Priority:  10,
		LinkIndex: 2,
	}

	tests := []struct {
		name     string
		r1       netlink.Route
		r2       netlink.Route
		expected bool
	}{
		{
			name:     "identical routes are equal",
			r1:       base,
			r2:       base,
			expected: true,
		},
		{
			name:     "different gateway is not equal",
			r1:       base,
			r2:       withGw(base, "10.0.0.9"),
			expected: false,
		},
		{
			name:     "different source is not equal",
			r1:       base,
			r2:       withSrc(base, "10.0.0.9"),
			expected: false,
		},
		{
			name:     "different priority is not equal",
			r1:       base,
			r2:       withPriority(base, 20),
			expected: false,
		},
		{
			name:     "different link index is not equal",
			r1:       base,
			r2:       withLinkIndex(base, 3),
			expected: false,
		},
	}

	p := &pipe{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.routeEqual(tt.r1, tt.r2)
			if got != tt.expected {
				t.Errorf("routeEqual() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func withGw(r netlink.Route, ip string) netlink.Route {
	r.Gw = net.ParseIP(ip)
	return r
}

func withSrc(r netlink.Route, ip string) netlink.Route {
	r.Src = net.ParseIP(ip)
	return r
}

func withPriority(r netlink.Route, prio int) netlink.Route {
	r.Priority = prio
	return r
}

func withLinkIndex(r netlink.Route, idx int) netlink.Route {
	r.LinkIndex = idx
	return r
}

func newTestPipe(t *testing.T) *pipe {
	t.Helper()

	pfx := mustParseCIDR(t, "10.0.0.0/8")
	return &pipe{
		name:        "test-pipe",
		prefix:      &pfx,
		sourceTable: 254,
		targetTable: 100,
		proto:       netlink.RouteProtocol(100),
		mu:          &sync.Mutex{},
	}
}

func TestPipeProcessUpdate_IgnoresUnrelatedTable(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "10.0.0.0/8")

	u := netlink.RouteUpdate{
		Type:  unix.RTM_NEWROUTE,
		Route: netlink.Route{Dst: &dst, Table: 999},
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.currentSource != nil || p.curentTarget != nil {
		t.Error("processUpdate() modified pipe state for an unrelated table")
	}
}

func TestPipeProcessUpdate_IgnoresNonMatchingPrefix(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "192.168.0.0/16")

	u := netlink.RouteUpdate{
		Type:  unix.RTM_NEWROUTE,
		Route: netlink.Route{Dst: &dst, Table: p.sourceTable},
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.currentSource != nil {
		t.Error("processUpdate() set currentSource for a non-matching prefix")
	}
}

func TestPipeProcessUpdate_AddInSourceEqualToTargetIsNoop(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "10.0.0.0/8")

	route := netlink.Route{Dst: &dst, Table: p.sourceTable, LinkIndex: 5}
	// Pretend the target table already has this exact route, so
	// processAddInSource() should short-circuit before touching netlink.
	p.curentTarget = &route

	u := netlink.RouteUpdate{
		Type:  unix.RTM_NEWROUTE,
		Route: route,
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.currentSource == nil {
		t.Fatal("processUpdate() did not record currentSource")
	}
}

func TestPipeProcessUpdate_AddInTargetOnlyTracksState(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "10.0.0.0/8")

	route := netlink.Route{Dst: &dst, Table: p.targetTable, LinkIndex: 7}
	u := netlink.RouteUpdate{
		Type:  unix.RTM_NEWROUTE,
		Route: route,
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.curentTarget == nil || p.curentTarget.LinkIndex != 7 {
		t.Errorf("curentTarget = %+v, want route with LinkIndex 7", p.curentTarget)
	}
}

func TestPipeProcessUpdate_RemoveInSourceClearsState(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "10.0.0.0/8")
	existing := netlink.Route{Dst: &dst}
	p.currentSource = &existing

	u := netlink.RouteUpdate{
		Type:  unix.RTM_DELROUTE,
		Route: netlink.Route{Dst: &dst, Table: p.sourceTable},
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.currentSource != nil {
		t.Error("processUpdate() did not clear currentSource on delete")
	}
}

func TestPipeProcessUpdate_RemoveInTargetWithDifferentProtoDoesNotSchedule(t *testing.T) {
	p := newTestPipe(t)
	dst := mustParseCIDR(t, "10.0.0.0/8")
	existing := netlink.Route{Dst: &dst}
	p.curentTarget = &existing

	// Protocol differs from p.proto, so processRemoveInTarget() must not
	// schedule a restore (which would eventually call replaceRoute()).
	u := netlink.RouteUpdate{
		Type:  unix.RTM_DELROUTE,
		Route: netlink.Route{Dst: &dst, Table: p.targetTable, Protocol: netlink.RouteProtocol(999)},
	}

	err := p.processUpdate(context.Background(), u)
	if err != nil {
		t.Fatalf("processUpdate() returned unexpected error: %v", err)
	}

	if p.curentTarget != nil {
		t.Error("processUpdate() did not clear curentTarget on delete")
	}
}
