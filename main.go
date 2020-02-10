package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const version = "0.2.0"

func main() {
	version := flag.Bool("version", false, "Shows version info")
	configFile := flag.String("config-file", "config.yml", "Path to config file")
	debug := flag.Bool("debug", false, "Enables debug output")
	metricsEndpoint := flag.String("metrics-endpoint", ":10080", "Endopint for scraping metrics")

	flag.Parse()

	if *version {
		printVersion()
		os.Exit(0)
	}

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	pipes, err := loadPipesFromConfig(*configFile)
	if err != nil {
		logrus.Fatal(err)
	}

	if len(pipes) == 0 {
		logrus.Info("No pipes defined.")
		os.Exit(0)
	}

	err = startMetricEndpoint(*metricsEndpoint)
	if err != nil {
		logrus.Fatal(err)
	}

	m := newMonitor(pipes)
	err = m.start()
	if err != nil {
		logrus.Fatal(err)
	}
}

func printVersion() {
	fmt.Println("piper")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk")
	fmt.Println("Copyright: 2020, Mauve Mailorder Software GmbH & Co. KG, Licensed under MIT license")
	fmt.Println("Piping routing information learned in a routing table to another one")
}

func loadPipesFromConfig(path string) ([]*pipe, error) {
	pipes := []*pipe{}

	cfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}

	for _, p := range cfg.Pipes {
		_, pfx, err := net.ParseCIDR(p.Prefix)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse %s", p.Prefix)
		}

		logrus.Infof("Configure pipe: %v", p)
		pipes = append(pipes, newPipe(p.Name, *pfx, p.Source, p.Target, cfg.Proto))
	}

	return pipes, nil
}
