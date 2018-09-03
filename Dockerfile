FROM alpine:3.6

RUN adduser -D jaeger-operator
USER jaeger-operator

ADD tmp/_output/bin/jaeger-operator /usr/local/bin/jaeger-operator
