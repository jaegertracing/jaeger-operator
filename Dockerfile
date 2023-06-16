# Build the manager binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.20 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY hack/install/install-dependencies.sh hack/install/
COPY hack/install/install-utils.sh hack/install/
COPY go.mod .
COPY go.sum .
RUN ./hack/install/install-dependencies.sh

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY cmd/ cmd/
COPY controllers/ controllers/
COPY pkg/ pkg/

COPY versions.txt versions.txt

ARG JAEGER_VERSION
ARG VERSION_PKG
ARG VERSION
ARG VERSION_DATE

# Dockerfile `FROM --platform=${BUILDPLATFORM}` means
# prepare image for build for matched BUILDPLATFORM, eq. linux/amd64
# by this way, we could avoid to using qemu, which slow down compiling process.
# and usefully for language who support multi-arch build like go.
# see last part of https://docs.docker.com/buildx/working-with-buildx/#build-multi-platform-images
ARG TARGETARCH
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -ldflags="-X ${VERSION_PKG}.version=${VERSION} -X ${VERSION_PKG}.buildDate=${VERSION_DATE} -X ${VERSION_PKG}.defaultJaeger=${JAEGER_VERSION}" -a -o jaeger-operator main.go

FROM registry.access.redhat.com/ubi8/ubi

ENV USER_UID=1001 \
    USER_NAME=jaeger-operator

RUN INSTALL_PKGS="openssl" && \
    yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \
    mkdir /tmp/_working_dir && \
    chmod og+w /tmp/_working_dir

WORKDIR /
COPY --from=builder /workspace/jaeger-operator .
COPY scripts/cert_generation.sh scripts/cert_generation.sh

USER ${USER_UID}:${USER_UID}

ENTRYPOINT ["/jaeger-operator"]
