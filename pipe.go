package main

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type pipe struct {
	name        string
	prefix      *net.IPNet
	sourceTable int
	targetTable int
	proto       int

	currentSource *netlink.Route
	curentTarget  *netlink.Route

	mu *sync.Mutex
}

func newPipe(name string, prefix net.IPNet, sourceTable int, targetTable int, proto int) *pipe {
	var pfx = &prefix

	o, _ := pfx.Mask.Size()
	if o == 0 {
		// default route
		pfx = nil
	}

	return &pipe{
		name:        name,
		prefix:      pfx,
		sourceTable: sourceTable,
		targetTable: targetTable,
		proto:       proto,
		mu:          &sync.Mutex{},
	}
}

func (p *pipe) processUpdate(ctx context.Context, u netlink.RouteUpdate) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if u.Table != p.sourceTable && u.Table != p.targetTable {
		return nil
	}

	if !p.pefixMatches(u.Dst) {
		return nil
	}

	defer recordRouteUpdateProcessed(ctx, &u, p)

	if u.Type == unix.RTM_DELROUTE {
		return p.processRemove(ctx, u)
	}

	return p.processAdd(ctx, u)
}

func (p *pipe) pefixMatches(pfx *net.IPNet) bool {
	if p.prefix == nil && pfx != nil {
		return false
	}

	if pfx == nil && p.prefix != nil {
		return false
	}

	if pfx == p.prefix {
		return true
	}

	return pfx.String() == p.prefix.String()
}

func (p *pipe) processAdd(ctx context.Context, u netlink.RouteUpdate) error {
	if u.Table == p.sourceTable {
		return p.processAddInSource(ctx, u)
	}

	return p.processAddInTarget(ctx, u)
}

func (p *pipe) processAddInSource(ctx context.Context, u netlink.RouteUpdate) error {
	logrus.Infof("Netlink added route in source table: %v", u.Route)
	p.currentSource = &u.Route

	if p.curentTarget != nil && p.routeEqual(*p.curentTarget, u.Route) {
		return nil
	}

	return p.replaceRoute(ctx, u.Route)
}

func (p *pipe) processAddInTarget(ctx context.Context, u netlink.RouteUpdate) error {
	logrus.Infof("Netlink added route in target table: %v", u.Route)
	p.curentTarget = &u.Route

	// nothing more to be done. we only want to set routes if no route for the prefix exists or its ours

	return nil
}

func (p *pipe) processRemove(ctx context.Context, u netlink.RouteUpdate) error {
	if u.Table == p.sourceTable {
		return p.processRemoveInSource(ctx, u)
	}

	return p.processRemoveInTarget(ctx, u)
}

func (p *pipe) processRemoveInSource(ctx context.Context, u netlink.RouteUpdate) error {
	logrus.Infof("Netlink removed route in source table: %v", u.Route)
	p.currentSource = nil
	return nil
}

func (p *pipe) processRemoveInTarget(ctx context.Context, u netlink.RouteUpdate) error {
	logrus.Infof("Netlink removed route in target table: %v", u.Route)
	p.curentTarget = nil

	if u.Protocol == p.proto {
		go func() {
			<-time.After(1 * time.Second)
			source := p.currentSource
			if source != nil && p.curentTarget == nil {
				logrus.Infof("Restoring route: &v", source)
				p.replaceRoute(ctx, *source)
			}
		}()
	}

	return nil
}

func (p *pipe) routeEqual(r1, r2 netlink.Route) bool {
	if !r1.Gw.Equal(r2.Gw) {
		return false
	}

	if !r1.Src.Equal(r2.Src) {
		return false
	}

	if r1.Priority != r2.Priority {
		return false
	}

	if r1.LinkIndex != r2.LinkIndex {
		return false
	}

	return true
}

func (p *pipe) replaceRoute(ctx context.Context, r netlink.Route) error {
	logrus.Infof("Replacing route: %v", r)

	new := &r
	new.Protocol = p.proto
	new.Table = p.targetTable
	err := netlink.RouteReplace(new)
	if err != nil {
		recordRouteReplaceError(ctx, p)
		return errors.Wrapf(err, "could not add route to table %d: %v", p.targetTable, r)
	}

	recordRouteReplaced(ctx, p)
	return nil
}
