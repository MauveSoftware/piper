package main

import (
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const proto int = 252

type pipe struct {
	prefix      *net.IPNet
	sourceTable int
	targetTable int

	currentSource *netlink.Route
	curentTarget  *netlink.Route
}

func newPipe(prefix net.IPNet, sourceTable int, targetTable int) *pipe {
	var pfx = &prefix

	o, _ := pfx.Mask.Size()
	if o == 0 {
		// default route
		pfx = nil
	}

	return &pipe{
		prefix:      pfx,
		sourceTable: sourceTable,
		targetTable: targetTable,
	}
}

func (p *pipe) processUpdate(u netlink.RouteUpdate) error {
	if u.Table != p.sourceTable && u.Table != p.targetTable {
		return nil
	}

	if !p.pefixMatches(u.Dst) {
		return nil
	}

	if u.Type == unix.RTM_DELROUTE {
		return p.processRemove(u)
	}

	return p.processAdd(u)
}

func (p *pipe) pefixMatches(pfx *net.IPNet) bool {
	if p.prefix == nil && pfx != nil {
		return false
	}

	if pfx == nil && p.prefix != nil {
		return false
	}

	return pfx.String() == p.prefix.String()
}

func (p *pipe) processAdd(u netlink.RouteUpdate) error {
	if u.Table == p.sourceTable {
		return p.processAddInSource(u)
	}

	return p.processAddInTarget(u)
}

func (p *pipe) processAddInSource(u netlink.RouteUpdate) error {
	logrus.Infof("Netlink added route in source table: %v", u.Route)
	p.currentSource = &u.Route

	if p.curentTarget != nil && p.routeEqual(*p.curentTarget, u.Route) {
		return nil
	}

	return p.replaceRoute(u.Route)
}

func (p *pipe) processAddInTarget(u netlink.RouteUpdate) error {
	logrus.Infof("Netlink added route in target table: %v", u.Route)
	p.curentTarget = &u.Route

	// nothing more to be done. we only want to set routes if no route for the prefix exists or its ours

	return nil
}

func (p *pipe) processRemove(u netlink.RouteUpdate) error {
	if u.Table == p.sourceTable {
		return p.processRemoveInSource(u)
	}

	return p.processRemoveInTarget(u)
}

func (p *pipe) processRemoveInSource(u netlink.RouteUpdate) error {
	logrus.Infof("Netlink removed route in source table: %v", u.Route)
	p.currentSource = nil
	return nil
}

func (p *pipe) processRemoveInTarget(u netlink.RouteUpdate) error {
	logrus.Infof("Netlink removed route in target table: %v", u.Route)
	p.curentTarget = nil

	if u.Protocol == proto {
		go func() {
			<-time.After(1 * time.Second)
			source := p.currentSource
			if source != nil && p.curentTarget == nil {
				logrus.Infof("Restoring route: &v", source)
				p.replaceRoute(*source)
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

	return true
}

func (p *pipe) replaceRoute(r netlink.Route) error {
	logrus.Infof("Replacing route: %v", r)

	new := &r
	new.Protocol = proto
	new.Table = p.targetTable
	err := netlink.RouteReplace(new)
	if err != nil {
		return errors.Wrapf(err, "could not add route to table %d: %v", p.targetTable, r)
	}

	return nil
}
