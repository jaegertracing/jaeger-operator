#!/bin/bash

NAMESPACE=${NAMESPACE:-"observability"} # same as the one in the Makefile
SERVICE=${SERVICE:-"jaeger-operator-webhook"}
POD=${POD:-"jaeger-operator"}
SECRET_NAME=${SECRET_NAME:-"jaeger-operator-webhook-cert"}

kubectl get secret "${SECRET_NAME}" >/dev/null 2>&1
if [ $? == 0 ]; then
    echo "The secret '${SECRET_NAME}' already exists. Skipping."
    exit 0
fi

command -v cfssl >/dev/null 2>&1
if [ $? != 0 ]; then
    echo "'cfssl' command not found. Run 'make install-tools' to install it. Aborting."
    exit 2
fi

echo "Generating CSR..."
cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": [
    "${SERVICE}.${NAMESPACE}.svc",
    "${SERVICE}.${NAMESPACE}.svc.cluster.local",
    "${POD}.${NAMESPACE}.pod",
    "${POD}.${NAMESPACE}.pod.cluster.local"
  ],
  "CN": "${POD}.${NAMESPACE}.pod.cluster.local",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
EOF

echo "Submitting CSR to Kubernetes..."
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${SERVICE}.${NAMESPACE}
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

echo "Approving CSR..."
kubectl certificate approve ${SERVICE}.${NAMESPACE}

echo "Storing signed certificate..."
kubectl get csr ${SERVICE}.${NAMESPACE} -o jsonpath='{.status.certificate}' | base64 --decode > server.crt

echo "Storing cert and key in the secret ${SECRET_NAME}..."
kubectl create secret tls ${SECRET_NAME} --key=server-key.pem --cert=server.crt -n ${NAMESPACE}

echo "Cleaning generated files..."
rm -f server.crt server.csr server-key.pem
