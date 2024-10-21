# The Dockerfile's resulting image is purpose-built for executing Jaeger Operator e2e tests within the OpenShift release (https://github.com/openshift/release) using Prow CI.
FROM golang@sha256:ad5c126b5cf501a8caef751a243bb717ec204ab1aa56dc41dc11be089fafcb4f

# Set the user to root
USER root

# Create the /tmp/go/bin directory, and grant read and write permissions to all users
RUN mkdir -p /tmp/go/bin \
    && chmod -R 777 /tmp/go/bin

# Set the build cache directory within /tmp and make it writable
ENV GOCACHE=/tmp/.cache/go-build
RUN mkdir -p $GOCACHE \
    && chmod -R 777 $GOCACHE

# Copy the repository files
COPY . /tmp/jaeger-operator

# Install kubectl and oc
RUN curl -L -o oc.tar.gz https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest/openshift-client-linux.tar.gz \
    && tar -xvzf oc.tar.gz \
    && chmod +x kubectl oc \
    && mv oc kubectl /usr/local/bin/ \
    && rm oc.tar.gz

# Set the working directory
WORKDIR /tmp/jaeger-operator

# Set the Go path
ENV GOPATH=/tmp/go
ENV GOBIN=/tmp/go/bin
ENV PATH=$PATH:$GOBIN

# Install required dependencies
RUN ./hack/install/install-kuttl.sh \
    && ./hack/install/install-golangci-lint.sh \
    && ./hack/install/install-goimports.sh \
    && ./hack/install/install-yq.sh \
    && ./hack/install/install-kustomize.sh \
    && ./hack/install/install-gomplate.sh \
    && ./hack/install/install-dependencies.sh \
    && ./hack/install/install-kubebuilder.sh \
    && ./hack/install/install-controller-gen.sh \
    && go install github.com/RH-QE-Distributed-Tracing/junitcli/cmd/junitcli@v1.0.6 \
    && cp ./bin/kubectl-kuttl /usr/local/bin/kubectl-kuttl

#Make required directories writable as Prow CI doesn't allow root user inside the container.
RUN chmod -R 777 /tmp/go /tmp/go/bin /tmp/.cache/
