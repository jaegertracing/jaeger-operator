# Sidecar - Skip webhook
## What is this test case testing?

When the `"app.kubernetes.io/name"="jaeger-operator"` label is added to a deployment,
the Jaeger agent is not added as a sidecar.

This test works in the following way:
* Create a Jaeger instance
* Enables the autoinjection of the Jaeger agent as a sidecar in a deployment but since the
label `"app.kubernetes.io/name"="jaeger-operator"` is set, the sidecar is not created
* Remove the label and check if the sidecar is created
