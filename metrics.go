package main

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	mRouteUpdatesProcessed = stats.Float64("route/updates_processed_count", "Number of route updates processed by pipe", "")
	mRouteUpdatesReceived  = stats.Float64("route/updates_received_count", "Number of route updates received from netlink", "")
	mRoutesReplaceSuccess  = stats.Float64("route/replace_success_count", "Number of piper route changes", "")
	mRoutesReplaceError    = stats.Float64("route/replace_error_count", "Number of failed piper route changes", "")

	keyPipeName, _        = tag.NewKey("pipe")
	keyRouteUpdateType, _ = tag.NewKey("type")
)

func startMetricEndpoint(listenAddress string) error {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "piper",
	})
	if err != nil {
		return errors.Wrap(err, "failed to create the Prometheus stats exporter")
	}

	err = view.Register(views()...)
	if err != nil {
		return errors.Wrap(err, "could not register views for Prometheus metrics")
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		logrus.Infof("Listening for metrics calls on /metrics at %s", listenAddress)
		if err := http.ListenAndServe(listenAddress, mux); err != nil {
			logrus.Errorf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()

	return nil
}

func views() []*view.View {
	return []*view.View{
		&view.View{
			Name:        mRouteUpdatesProcessed.Name(),
			Description: mRouteUpdatesProcessed.Description(),
			Aggregation: view.Count(),
			Measure:     mRouteUpdatesProcessed,
		},
		&view.View{
			Name:        mRouteUpdatesReceived.Name(),
			Description: mRouteUpdatesReceived.Description(),
			Aggregation: view.Count(),
			Measure:     mRouteUpdatesReceived,
		},
		&view.View{
			Name:        mRoutesReplaceSuccess.Name(),
			Description: mRoutesReplaceSuccess.Description(),
			Aggregation: view.Count(),
			Measure:     mRoutesReplaceSuccess,
		},
		&view.View{
			Name:        mRoutesReplaceError.Name(),
			Description: mRoutesReplaceError.Description(),
			Aggregation: view.Count(),
			Measure:     mRoutesReplaceError,
		},
	}
}

func recordRouteUpdateReceived(ctx context.Context, u *netlink.RouteUpdate) {
	ctx, _ = tag.New(ctx, tag.Insert(keyRouteUpdateType, updateType(u)))
	stats.Record(ctx, mRouteUpdatesReceived.M(1))
}

func recordRouteUpdateProcessed(ctx context.Context, u *netlink.RouteUpdate, p *pipe) {
	ctx, _ = tag.New(ctx, tag.Insert(keyRouteUpdateType, updateType(u)), tag.Insert(keyPipeName, p.name))
	stats.Record(ctx, mRouteUpdatesProcessed.M(1))
}

func recordRouteReplaced(ctx context.Context, p *pipe) {
	ctx, _ = tag.New(ctx, tag.Insert(keyPipeName, p.name))
	stats.Record(ctx, mRoutesReplaceSuccess.M(1))
}

func recordRouteReplaceError(ctx context.Context, p *pipe) {
	ctx, _ = tag.New(ctx, tag.Insert(keyPipeName, p.name))
	stats.Record(ctx, mRoutesReplaceError.M(1))
}

func updateType(u *netlink.RouteUpdate) string {
	if u.Type == unix.RTM_DELROUTE {
		return "delete"
	}

	return "add"
}
