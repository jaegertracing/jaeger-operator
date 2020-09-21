FROM golang:1.14 as builder

ARG GOPROXY
ENV GOPROXY=${GOPROXY}

COPY . /go/src/github.com/jaegertracing/jaeger-operator/
WORKDIR /go/src/github.com/jaegertracing/jaeger-operator
RUN make gobuild

FROM registry.access.redhat.com/ubi8/ubi

ENV OPERATOR=/usr/local/bin/jaeger-operator \
    USER_UID=1001 \
    USER_NAME=jaeger-operator

RUN INSTALL_PKGS=" \
      openssl \
      " && \
    yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \
    mkdir /tmp/_working_dir && \
    chmod og+w /tmp/_working_dir

COPY --from=builder /go/src/github.com/jaegertracing/jaeger-operator/scripts/* /scripts/

# install operator binary
COPY --from=builder /go/src/github.com/jaegertracing/jaeger-operator/build/_output/bin/jaeger-operator ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/jaeger-operator"]

USER ${USER_UID}
