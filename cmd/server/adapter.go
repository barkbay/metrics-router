package server

import (
	"context"
	"flag"
	"os"
	"os/user"
	"path"

	"github.com/barkbay/custom-metrics-router/pkg/controllers/metricsource"
	"github.com/barkbay/custom-metrics-router/pkg/provider"
	basecmd "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/cmd"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

type RoutedAdapter struct {
	basecmd.AdapterBase
	*metricsource.Registry
}

func NewRoutedAdapter(
	registry *metricsource.Registry,
	flags *pflag.FlagSet,
) *RoutedAdapter {
	router := &RoutedAdapter{
		Registry: registry,
	}
	router.Flags().AddFlagSet(flags)
	return router
}

func (r *RoutedAdapter) run(ctx context.Context) {
	r.Flags().AddGoFlagSet(flag.CommandLine) // make sure you get the klog flags
	err := r.Flags().Parse(os.Args)
	if err != nil {
		klog.Fatalf("failed to parse flags: %v", err)
	}
	routedProvider := provider.NewRoutedProvider(r.Registry)
	r.WithCustomMetrics(routedProvider)
	r.WithExternalMetrics(routedProvider)

	// Attempt to load the config
	if _, err := r.ClientConfig(); err != rest.ErrNotInCluster {
		// Not in cluster, attempt to laad the config from a known place
		kubeconfigPath := detectKubeconfigPath()
		if kubeconfigPath == "" {
			klog.Fatalf(
				"failed to load cluster configuration, use env variable %s to set a path to set a path to configuration file",
				clientcmd.RecommendedConfigPathEnvVar,
			)
		}
		r.RemoteKubeConfigFile = kubeconfigPath
		r.Authorization.RemoteKubeConfigFile = kubeconfigPath
		r.Authentication.RemoteKubeConfigFile = kubeconfigPath
	}

	if viper.GetBool("anonymous-auth") {
		r.Authentication = nil
		r.Authorization = nil
	}

	if err := r.Run(ctx.Done()); err != nil {
		klog.Fatalf("unable to run custom metrics routedProvider: %v", err)
	}
}

func detectKubeconfigPath() string {
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) != 0 {
		return kubeconfigPath
	}
	if _, ok := os.LookupEnv("HOME"); ok {
		u, err := user.Current()
		if err != nil {
			klog.V(2).Infof("Unable to detect home for user: %s", err)
			return ""
		}
		return path.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
	}
	return ""
}
