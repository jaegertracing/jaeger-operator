package ingress

import (
	"context"

	"github.com/spf13/viper"

	extv1beta "k8s.io/api/extensions/v1beta1"
	netv1beta "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtensionAPI indicate k8s networking.k8s.io/v1beta1 should be used on the current cluster
const ExtensionAPI = "extension"

// NetworkingAPI indicate k8s extensions/v1beta1 should be used on the current cluster
const NetworkingAPI = "networking"

// Client wrap around k8s client, and decide which ingress API should be used, depending on cluster capabilities.
type Client struct {
	client       client.Client
	rClient      client.Reader
	extensionAPI bool
}

// NewIngressClient Creates a new Ingress.client wrapper.
func NewIngressClient(client client.Client, reader client.Reader) *Client {
	return &Client{
		client:       client,
		rClient:      reader,
		extensionAPI: viper.GetString("ingress-api") == ExtensionAPI,
	}
}

func (c *Client) fromExtToNet(ingress extv1beta.Ingress) netv1beta.Ingress {
	oldIngress := netv1beta.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: ingress.ObjectMeta,
	}

	if ingress.Spec.Backend != nil {
		oldIngress.Spec = netv1beta.IngressSpec{
			Backend: &netv1beta.IngressBackend{
				ServiceName: ingress.Spec.Backend.ServiceName,
				ServicePort: ingress.Spec.Backend.ServicePort,
			},
		}
	}

	for _, tls := range ingress.Spec.TLS {
		oldIngress.Spec.TLS = append(oldIngress.Spec.TLS, netv1beta.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		})
	}

	for _, rule := range ingress.Spec.Rules {
		httpIngressPaths := make([]netv1beta.HTTPIngressPath, len(rule.HTTP.Paths))
		for i, path := range rule.HTTP.Paths {
			httpIngressPaths[i].Backend.ServicePort = path.Backend.ServicePort
			httpIngressPaths[i].Backend.ServiceName = path.Backend.ServiceName
			httpIngressPaths[i].Path = path.Path

		}

		oldIngress.Spec.Rules = append(oldIngress.Spec.Rules, netv1beta.IngressRule{
			Host: rule.Host,
			IngressRuleValue: netv1beta.IngressRuleValue{
				HTTP: &netv1beta.HTTPIngressRuleValue{
					Paths: httpIngressPaths,
				},
			},
		})
	}

	return oldIngress
}

func (c *Client) fromNetToExt(ingress netv1beta.Ingress) extv1beta.Ingress {
	oldIngress := extv1beta.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: ingress.ObjectMeta,
	}

	if ingress.Spec.Backend != nil {
		oldIngress.Spec = extv1beta.IngressSpec{
			Backend: &extv1beta.IngressBackend{
				ServiceName: ingress.Spec.Backend.ServiceName,
				ServicePort: ingress.Spec.Backend.ServicePort,
			},
		}
	}

	for _, tls := range ingress.Spec.TLS {
		oldIngress.Spec.TLS = append(oldIngress.Spec.TLS, extv1beta.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		})
	}

	for _, rule := range ingress.Spec.Rules {
		httpIngressPaths := make([]extv1beta.HTTPIngressPath, len(rule.HTTP.Paths))
		for i, path := range rule.HTTP.Paths {
			httpIngressPaths[i].Backend.ServicePort = path.Backend.ServicePort
			httpIngressPaths[i].Backend.ServiceName = path.Backend.ServiceName
			httpIngressPaths[i].Path = path.Path

		}

		oldIngress.Spec.Rules = append(oldIngress.Spec.Rules, extv1beta.IngressRule{
			Host: rule.Host,
			IngressRuleValue: extv1beta.IngressRuleValue{
				HTTP: &extv1beta.HTTPIngressRuleValue{
					Paths: httpIngressPaths,
				},
			},
		})
	}

	return oldIngress
}

// List is a wrap function that calls k8s client List with extend or networking API.
func (c *Client) List(ctx context.Context, list *netv1beta.IngressList, opts ...client.ListOption) error {
	if c.extensionAPI {
		extIngressList := extv1beta.IngressList{}
		err := c.rClient.List(ctx, &extIngressList, opts...)
		if err != nil {
			return err
		}
		for _, item := range extIngressList.Items {
			list.Items = append(list.Items, c.fromExtToNet(item))
		}
		return nil
	}
	return c.rClient.List(ctx, list, opts...)
}

// Update is a wrap function that calls k8s client Update with extend or networking API.
func (c *Client) Update(ctx context.Context, obj *netv1beta.Ingress, opts ...client.UpdateOption) error {
	if c.extensionAPI {
		extIngressList := c.fromNetToExt(*obj)
		return c.client.Update(ctx, &extIngressList, opts...)
	}
	return c.client.Update(ctx, obj, opts...)

}

// Delete is a wrap function that calls k8s client Delete with extend or networking API.
func (c *Client) Delete(ctx context.Context, obj *netv1beta.Ingress, opts ...client.DeleteOption) error {
	if c.extensionAPI {
		extIngressList := c.fromNetToExt(*obj)
		return c.client.Delete(ctx, &extIngressList, opts...)
	}
	return c.client.Delete(ctx, obj, opts...)
}

// Create is a wrap function that calls k8s client Create with extend or networking API.
func (c *Client) Create(ctx context.Context, obj *netv1beta.Ingress, opts ...client.CreateOption) error {
	if c.extensionAPI {
		extIngressList := c.fromNetToExt(*obj)
		return c.client.Create(ctx, &extIngressList, opts...)
	}
	return c.client.Create(ctx, obj, opts...)
}
