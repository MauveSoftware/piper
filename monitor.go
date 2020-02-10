package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)

	logrus.Info("Listening for routing update")

	for {
		select {
		case u := <-ch:
			m.processUpdate(u)
		case <-term:
			logrus.Info("Shutting down gracefully. Unsubscribing from netlink.")
			close(done)
			return nil
		}
	}
}

func (m *monitor) processUpdate(u netlink.RouteUpdate) {
	logrus.Debug("Got route update", u)

	ctx := context.Background()
	recordRouteUpdateReceived(ctx, &u)

	for _, p := range m.pipes {
		logrus.Debug("Processing pipe: ", p)
		err := p.processUpdate(ctx, u)
		if err != nil {
			logrus.Errorf("Error on route update: %v", err)
		}
	}
}
