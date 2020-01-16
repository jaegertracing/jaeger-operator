// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1beta1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.Kafka":                schema_pkg_apis_kafka_v1beta1_Kafka(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaSpec":            schema_pkg_apis_kafka_v1beta1_KafkaSpec(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatus":          schema_pkg_apis_kafka_v1beta1_KafkaStatus(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatusCondition": schema_pkg_apis_kafka_v1beta1_KafkaStatusCondition(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUser":            schema_pkg_apis_kafka_v1beta1_KafkaUser(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserSpec":        schema_pkg_apis_kafka_v1beta1_KafkaUserSpec(ref),
		"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserStatus":      schema_pkg_apis_kafka_v1beta1_KafkaUserStatus(ref),
	}
}

func schema_pkg_apis_kafka_v1beta1_Kafka(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Kafka is the Schema for the kafkas API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaSpec", "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaSpec defines the desired state of Kafka",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"json": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "byte",
						},
					},
				},
				Required: []string{"json"},
			},
		},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaStatus defines the observed state of Kafka",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"conditions": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatusCondition"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatusCondition"},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaStatusCondition(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaStatusCondition holds the different conditions affecting the Kafka instance",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"lastTransitionTime": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"reason": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"message": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaUser(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaUser is the Schema for the kafkausers API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserSpec", "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaUserStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaUserSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaUserSpec defines the desired state of KafkaUser",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"json": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "byte",
						},
					},
				},
				Required: []string{"json"},
			},
		},
	}
}

func schema_pkg_apis_kafka_v1beta1_KafkaUserStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KafkaUserStatus defines the observed state of KafkaUser",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"conditions": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatusCondition"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta1.KafkaStatusCondition"},
	}
}
