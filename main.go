package main

import (
	"context"
	"flag"
	"os"

	"k8s.io/klog"

	"github.com/xwen-winnie/crd_demo/kube"
	"github.com/xwen-winnie/crd_demo/kube/signals"
)

var (
	f string
)

func init() {
	flag.StringVar(&f, "f", "etc/config.yaml", "the configuration file")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	cli, err := kube.NewClient(context.Background(), f)
	if err != nil {
		klog.Fatal(err.Error())
		os.Exit(-1)
	}
	cli.Start(stopCh)

	err = cli.Run(2, stopCh)
	if err != nil {
		klog.Fatal(err.Error())
		os.Exit(-1)
	}
}
