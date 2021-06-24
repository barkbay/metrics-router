# Metrics Router

Metrics router is a router for Custom and External Metrics. It allows the use of multiples metric servers as sources for the K8S autoscaling controllers.

> :warning: **This project should be considered as a proof of concept** used to experiment around the ideas proposed in this [KEP](https://github.com/kubernetes/enhancements/pull/2581).

## Install

### Installing the CRDs and the server

```
% kubectl apply -f https://raw.githubusercontent.com/barkbay/metrics-router/main/config/deploy/all-in-one.yaml
```

Check that the metrics server is up and running:

```json
% kubectl get pods
NAME                                                 READY   STATUS    RESTARTS   AGE
metrics-router-controller-manager-5474f885c5-jtwmw   2/2     Running   0          53
```

### Configuring the APIService

```
% kubectl apply -f https://raw.githubusercontent.com/barkbay/metrics-router/main/config/deploy/api-service.yaml
```

## Creating a metrics source

```
cat <<EOF | kubectl apply -f -
apiVersion: metricsrouter.io.metricsrouter.io/v1alpha1
kind: MetricsSource
metadata:
  name: prometheus
spec:
  insecureSkipTLSVerify: true
  priority: 100
  metricTypes:
    - CustomMetrics
  service:
    namespace: custom-metrics
    name: prometheus-metrics-apiserver
    port:
      number: 443
EOF
```

You can check if the resource has been loaded using `kubectl get ms`:

``` 
% kubectl get ms
NAME         SERVICE                                       PORT   SYNCED   METRICS
prometheus   custom-metrics/prometheus-metrics-apiserver   443    true     476
```

* `SYNCED` reports if the metrics list has been successfully retrieved from the metrics source backend.
* The number of metrics loaded is displayed in the `METRICS` columns.

## Metrics sources prioritization

If a metric is served by more than one backend, the metrics source with the higher `priority` is used. The higher the value, the higher the priority. Having two metrics sources with the same priority should be avoided, in such a case the metrics sources are sorted by name.

## Troubleshooting

### Getting metrics server logs

```json
% kubectl logs -l control-plane=controller-manager -c server -f
I0624 07:11:55.995653       1 controller.go:70] syncing metrics from /prometheus
I0624 07:11:55.995728       1 registry.go:67] Update metrics source prometheus
I0624 07:11:56.027753       1 controller.go:94] 61 metrics loaded from /prometheus
I0624 07:12:19.155079       1 registry.go:201] custom metric pods/foo(namespaced) served by https://custom-metrics-apiserver.custom-metrics.svc:443
I0624 07:12:19.155114       1 metricsclient.go:190] custom metric info: provider.CustomMetricInfo{GroupResource:schema.GroupResource{Group:"", Resource:"pods"}, Namespaced:true, Metric:"foo"}
```

### Getting HPA events

HPA events can provide useful information to understand why metrics are not retrieved, for example using the `describe` subcommand:

```
% kubectl describe hpa/my-hpa -n my-ns
Name:                                             my-hpa 
Namespace:                                        my-ns
...
Events:
  Type     Reason                        Age                  From                       Message
  ----     ------                        ----                 ----                       -------
  Warning  FailedComputeMetricsReplicas  11s                  horizontal-pod-autoscaler  failed to compute desired number of replicas based on listed metrics for Resource/ns/resource: invalid metrics (1 invalid out of 1), first error is: failed to get pods metric value: unable to get metric foo: unable to fetch metrics from custom metrics API: custom metric foo is not provided by any metrics backend
```

## Credits

The core algorithm is a fork from https://github.com/arjunrn/custom-metrics-router

