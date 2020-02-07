package main

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type monitor struct {
	pipes []*pipe
}

func newMonitor(pipes []*pipe) *monitor {
	return &monitor{
		pipes: pipes,
	}
}

func (m *monitor) start() error {
	ch := make(chan netlink.RouteUpdate)
	done := make(chan struct{})

	logrus.Info("Subscribing for routing update")
	opt := netlink.RouteSubscribeOptions{
		ListExisting: true,
	}
	err := netlink.RouteSubscribeWithOptions(ch, done, opt)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to netlink for route changes")
	}
	defer close(done)

	logrus.Info("Listening for routing update")
	for u := range ch {
		m.processUpdate(u)
	}

	return nil
}

func (m *monitor) processUpdate(u netlink.RouteUpdate) {
	logrus.Debug("Got route update", u)

	for _, p := range m.pipes {
		err := p.processUpdate(u)
		if err != nil {
			logrus.Errorf("Error on route update: %v", err)
		}
	}
}
