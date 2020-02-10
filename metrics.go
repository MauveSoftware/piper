package main

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	mUp                    = stats.Float64("up", "Returns 1 if piper is running", "")
	mRouteUpdatesProcessed = stats.Float64("route/update/processed_count", "Number of route updates processed by pipe", "")
	mRouteUpdatesReceived  = stats.Float64("route/update/received_count", "Number of route updates received from netlink", "")
	mRoutesReplaceSuccess  = stats.Float64("route/replace/success_count", "Number of piper route changes", "")
	mRoutesReplaceError    = stats.Float64("route/replace/error_count", "Number of failed piper route changes", "")

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

	stats.Record(context.Background(), mUp.M(1))

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
			Name:        mUp.Name(),
			Description: mUp.Description(),
			Aggregation: view.LastValue(),
			Measure:     mUp,
		},
		&view.View{
			Name:        mRouteUpdatesProcessed.Name(),
			Description: mRouteUpdatesProcessed.Description(),
			Aggregation: view.Count(),
			Measure:     mRouteUpdatesProcessed,
			TagKeys:     []tag.Key{keyPipeName, keyRouteUpdateType},
		},
		&view.View{
			Name:        mRouteUpdatesReceived.Name(),
			Description: mRouteUpdatesReceived.Description(),
			Aggregation: view.Count(),
			Measure:     mRouteUpdatesReceived,
			TagKeys:     []tag.Key{keyRouteUpdateType},
		},
		&view.View{
			Name:        mRoutesReplaceSuccess.Name(),
			Description: mRoutesReplaceSuccess.Description(),
			Aggregation: view.Count(),
			Measure:     mRoutesReplaceSuccess,
			TagKeys:     []tag.Key{keyPipeName},
		},
		&view.View{
			Name:        mRoutesReplaceError.Name(),
			Description: mRoutesReplaceError.Description(),
			Aggregation: view.Count(),
			Measure:     mRoutesReplaceError,
			TagKeys:     []tag.Key{keyPipeName},
		},
	}
}

func recordRouteUpdateReceived(ctx context.Context, u *netlink.RouteUpdate) {
	stats.RecordWithTags(ctx, []tag.Mutator{
		tag.Insert(keyRouteUpdateType, routeUpdateType(u)),
	}, mRouteUpdatesReceived.M(1))
}

func recordRouteUpdateProcessed(ctx context.Context, u *netlink.RouteUpdate, p *pipe) {
	stats.RecordWithTags(ctx, []tag.Mutator{
		tag.Insert(keyPipeName, p.name),
		tag.Insert(keyRouteUpdateType, routeUpdateType(u)),
	}, mRouteUpdatesProcessed.M(1))
}

func recordRouteReplaced(ctx context.Context, p *pipe) {
	stats.RecordWithTags(ctx, []tag.Mutator{
		tag.Insert(keyPipeName, p.name),
	}, mRoutesReplaceSuccess.M(1))
}

func recordRouteReplaceError(ctx context.Context, p *pipe) {
	stats.RecordWithTags(ctx, []tag.Mutator{
		tag.Insert(keyPipeName, p.name),
	}, mRoutesReplaceError.M(1))
}
