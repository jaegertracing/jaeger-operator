package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// CreatePortForward listens for local connections and forwards them to a remote pod
func CreatePortForward(namespace, podNamePrefix, containsImage string, ports []string, kConfig *rest.Config) (*portforward.PortForwarder, chan struct{}) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(kConfig)
	if err != nil {
		printTestStackTrace()
		require.NoError(t, err)
	}

	pod := GetPod(namespace, podNamePrefix, containsImage, fw.KubeClient)
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, pod.Name)
	hostIP := strings.TrimLeft(kConfig.Host, "https://")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		printTestStackTrace()
		require.NoError(t, err)
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
		}
		if len(errOut.String()) != 0 {
			panic(errOut.String())
		} else if len(out.String()) != 0 {
			fmt.Println(out.String())
		}
	}()
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			panic(err)
		}
	}()
	<- forwarder.Ready
	return forwarder, stopChan
}
