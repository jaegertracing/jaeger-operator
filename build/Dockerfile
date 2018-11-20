FROM alpine:3.8

USER nobody

ADD build/_output/bin/jaeger-operator /usr/local/bin/jaeger-operator
ENTRYPOINT ["/usr/local/bin/jaeger-operator"]
