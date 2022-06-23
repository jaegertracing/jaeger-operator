# Sidecar - Namespace
## What is this test case testing?

Test the deployment of the Jaeger agent as as sidecar annotating a namespace.

[From the Jaeger documentation](https://www.jaegertracing.io/docs/latest/operator/#auto-injecting-jaeger-agent-sidecars):
> The operator can inject Jaeger Agent sidecars in Deployment workloads,
provided that the deployment or its namespace has the annotation
`sidecar.jaegertracing.io/inject` with a suitable value. The values can be either
"true" (as string), or the Jaeger instance name, as returned by `kubectl get
jaegers`. When "true" is used, there should be exactly one Jaeger instance for
the same namespace as the deployment, otherwise, the operator can’t figure out
automatically which Jaeger instance to use. A specific Jaeger instance name on
a deployment has a higher precedence than true applied on its namespace.

This test works in the following way:
* Create a Jaeger instance
* Inject the Jaeger agent as a sidecard for a deployment
* Create a second Jaeger instance
* Remove the first Jaeger instance (now, the deployment will use the second Jaeger instance)
* Check there were no errors
* Check the sidecar is removed when the annotation is removed
