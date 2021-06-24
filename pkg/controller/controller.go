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

package controller

import (
	"context"
	"reflect"
	"time"

	"github.com/barkbay/custom-metrics-router/pkg/registry"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mrv1alpha1 "github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
)

func SetupMetricsSourceController(mgr ctrl.Manager) (*registry.Registry, error) {
	k8sClient := mgr.GetClient()

	// Create a new routes registry
	registry := registry.NewRegistry(mgr.GetConfig(), k8sClient.RESTMapper())

	// Create the reconciler
	reconciler := &MetricsSourceReconciler{
		Client:   k8sClient,
		Scheme:   mgr.GetScheme(),
		registry: registry,
	}

	// Register the reconciler
	return registry, reconciler.SetupWithManager(mgr)
}

// MetricsSourceReconciler reconciles a MetricsSource object
type MetricsSourceReconciler struct {
	client.Client
	registry *registry.Registry
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=metricsrouter.io.metricsrouter.io,resources=metricssources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metricsrouter.io.metricsrouter.io,resources=metricssources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=metricsrouter.io.metricsrouter.io,resources=metricssources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MetricsSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	klog.Infof("syncing metrics from %s", req)
	// Get the Metrics source
	metricsSource := &mrv1alpha1.MetricsSource{}
	err := r.Client.Get(context.Background(), req.NamespacedName, metricsSource)
	if errors.IsNotFound(err) || metricsSource.IsMarkedForDeletion() {
		r.registry.DeleteSource(req.Name)
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	metricCount, err := r.registry.AddOrUpdateSource(*metricsSource)
	newStatus := mrv1alpha1.MetricsSourceStatus{
		Synced:       err == nil,
		MetricsCount: metricCount,
		Service:      metricsSource.Spec.MetricsServiceBackend.NamespacedName().String(),
		Port:         int(metricsSource.Spec.MetricsServiceBackend.Port.Port()),
	}
	// Always attempt to update the status
	if err != nil {
		_ = r.updateStatus(metricsSource, newStatus)
		return ctrl.Result{}, err
	}
	klog.Infof("%d metrics loaded from %s", metricCount, req)
	return ctrl.Result{
		RequeueAfter: 5 * time.Minute, // reload metric list every 5 minutes by default
	}, r.updateStatus(metricsSource, newStatus)
}

func (r *MetricsSourceReconciler) updateStatus(metricsSource *mrv1alpha1.MetricsSource, newStatus mrv1alpha1.MetricsSourceStatus) error {
	if reflect.DeepEqual(metricsSource.Status, newStatus) {
		return nil
	}
	metricsSource.Status = newStatus
	return r.Client.Status().Update(context.Background(), metricsSource)
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetricsSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mrv1alpha1.MetricsSource{}).
		Complete(r)
}
