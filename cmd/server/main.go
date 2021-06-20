/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	mrv1alpha1 "github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"github.com/barkbay/custom-metrics-router/pkg/controllers/metricsource"

	_ "gopkg.in/yaml.v2"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mrv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Metric Router server",
		Run:   doRun,
	}
	cmd.Flags().Bool("anonymous-auth", false, "if true, metrics server authentication and authorization are disabled, only to be used in dev mode")
	cmd.Flags().String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	return cmd
}

func doRun(cmd *cobra.Command, _ []string) {
	/*devMode := viper.GetBool("development")
	opts := zap.Options{
		Development: devMode,
	}
	opts.BindFlags(flag.CommandLine)*/
	flag.Parse()
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		setupLog.Error(err, "failed to bind flags")
		os.Exit(1)
	}

	//ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     viper.GetString("metrics-bind-address"),
		Port:                   9443,
		HealthProbeBindAddress: viper.GetString("health-probe-bind-address"),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create a new routes registry
	registry, err := metricsource.SetupMetricsSourceController(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MetricsSource")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()
	go NewRoutedAdapter(registry, cmd.Flags()).run(signalHandler)

	setupLog.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
