FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.16 as builder

ARG JAEGER_VERSION
ENV JAEGER_VERSION=${JAEGER_VERSION}

COPY . /go/src/github.com/jaegertracing/jaeger-operator/
WORKDIR /go/src/github.com/jaegertracing/jaeger-operator

ARG GOPROXY
# download deps before gobuild
RUN go mod download -x

# Dockerfile `FROM --platform=${BUILDPLATFORM}` means
# prepare image for build for matched BUILDPLATFORM, eq. linux/amd64
# by this way, we could avoid to using qemu, which slow down compiling process.
# and usefully for language who support multi-arch build like go.
# see last part of https://docs.docker.com/buildx/working-with-buildx/#build-multi-platform-images
ARG TARGETARCH
# when --platform=linux/amd64,linux/arm64
#
# for $TARGETARCH in "amd64 arm64" do
RUN make gobuild OUTPUT_BINARY=/go/bin/jaeger-operator-${TARGETARCH} GOARCH=${TARGETARCH}
# done

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
ARG TARGETARCH
COPY --from=builder /go/bin/jaeger-operator-${TARGETARCH} ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/jaeger-operator"]

USER ${USER_UID}:${USER_UID}
