FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base

RUN INSTALL_PKGS=" \
      openssl \
      " && \
    yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \
    mkdir /tmp/_working_dir && \
    chmod og+w /tmp/_working_dir
COPY scripts/* /usr/bin/scripts/

USER nobody

ADD build/_output/bin/jaeger-operator /usr/local/bin/jaeger-operator
ENTRYPOINT ["/usr/local/bin/jaeger-operator"]
