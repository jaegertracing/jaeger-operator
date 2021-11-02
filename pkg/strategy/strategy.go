package strategy

import (
	osconsolev1 "github.com/openshift/api/console/v1"
	osv1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/consolelink"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

// S knows what type of deployments to build based on a given spec
type S struct {
	typ v1.DeploymentStrategy
	// When adding a new type here, remember to update All() too
	accounts                 []corev1.ServiceAccount
	clusterRoleBindings      []rbac.ClusterRoleBinding
	configMaps               []corev1.ConfigMap
	consoleLinks             []osconsolev1.ConsoleLink
	cronJobs                 []batchv1beta1.CronJob
	daemonSets               []appsv1.DaemonSet
	dependencies             []batchv1.Job
	deployments              []appsv1.Deployment
	elasticsearches          []esv1.Elasticsearch
	horizontalPodAutoscalers []autoscalingv2beta2.HorizontalPodAutoscaler
	ingresses                []networkingv1.Ingress
	kafkas                   []kafkav1beta2.Kafka
	kafkaUsers               []kafkav1beta2.KafkaUser
	routes                   []osv1.Route
	services                 []corev1.Service
	secrets                  []corev1.Secret
}

// New constructs a new strategy from scratch
func New() *S {
	return &S{}
}

// Type returns the strategy type for the given strategy
func (s S) Type() v1.DeploymentStrategy {
	return s.typ
}

// WithAccounts returns the strategy with the given list of service accounts
func (s S) WithAccounts(accs []corev1.ServiceAccount) S {
	s.accounts = accs
	return s
}

// WithClusterRoleBindings returns the strategy with the given list of config maps
func (s S) WithClusterRoleBindings(c []rbac.ClusterRoleBinding) S {
	s.clusterRoleBindings = c
	return s
}

// WithConsoleLinks returns the strategy with the given list of console links
func (s S) WithConsoleLinks(c []osconsolev1.ConsoleLink) S {
	s.consoleLinks = c
	return s
}

// WithConfigMaps returns the strategy with the given list of config maps
func (s S) WithConfigMaps(c []corev1.ConfigMap) S {
	s.configMaps = c
	return s
}

// WithCronJobs returns the strategy with the given list of cron jobs
func (s S) WithCronJobs(c []batchv1beta1.CronJob) S {
	s.cronJobs = c
	return s
}

// WithDeployments returns the strategy with the given list of deployments
func (s S) WithDeployments(deps []appsv1.Deployment) S {
	s.deployments = deps
	return s
}

// WithDaemonSets returns the strategy with the given list of daemonsets
func (s S) WithDaemonSets(d []appsv1.DaemonSet) S {
	s.daemonSets = d
	return s
}

// WithDependencies returns the strategy with the given list of dependencies
func (s S) WithDependencies(deps []batchv1.Job) S {
	s.dependencies = deps
	return s
}

// WithElasticsearches returns the strategy with the given list of elastic search instances
func (s S) WithElasticsearches(es []esv1.Elasticsearch) S {
	s.elasticsearches = es
	return s
}

// WithIngresses returns the strategy with the given list of dependencies
func (s S) WithIngresses(i []networkingv1.Ingress) S {
	s.ingresses = i
	return s
}

// WithHorizontalPodAutoscaler returns the strategy with the given list of HPAs
func (s S) WithHorizontalPodAutoscaler(i []autoscalingv2beta2.HorizontalPodAutoscaler) S {
	s.horizontalPodAutoscalers = i
	return s
}

// WithRoutes returns the strategy with the given list of routes
func (s S) WithRoutes(r []osv1.Route) S {
	s.routes = r
	return s
}

// WithKafkas returns the strategy with the given list of Kafkas
func (s S) WithKafkas(k []kafkav1beta2.Kafka) S {
	s.kafkas = k
	return s
}

// WithKafkaUsers returns the strategy with the given list of Kafka Users
func (s S) WithKafkaUsers(k []kafkav1beta2.KafkaUser) S {
	s.kafkaUsers = k
	return s
}

// WithServices returns the strategy with the given list of routes
func (s S) WithServices(svcs []corev1.Service) S {
	s.services = svcs
	return s
}

// WithSecrets returns the strategy with the given list of secrets
func (s S) WithSecrets(secrets []corev1.Secret) S {
	s.secrets = secrets
	return s
}

// Accounts returns the list of service accounts for this strategy
func (s S) Accounts() []corev1.ServiceAccount {
	return s.accounts
}

// ClusterRoleBindings returns the list of cluster role bindings for this strategy
func (s S) ClusterRoleBindings() []rbac.ClusterRoleBinding {
	return s.clusterRoleBindings
}

// ConfigMaps returns the list of config maps for this strategy
func (s S) ConfigMaps() []corev1.ConfigMap {
	return s.configMaps
}

// ConsoleLinks returns the console links for this strategy
func (s S) ConsoleLinks(routes []osv1.Route) []osconsolev1.ConsoleLink {
	return consolelink.UpdateHref(routes, s.consoleLinks)
}

// CronJobs returns the list of cron jobs for this strategy
func (s S) CronJobs() []batchv1beta1.CronJob {
	return s.cronJobs
}

// DaemonSets returns the list of daemon sets for this strategy
func (s S) DaemonSets() []appsv1.DaemonSet {
	return s.daemonSets
}

// Deployments returns the list of deployments for this strategy
func (s S) Deployments() []appsv1.Deployment {
	return s.deployments
}

// Elasticsearches returns the list of elastic search instances for this strategy
func (s S) Elasticsearches() []esv1.Elasticsearch {
	return s.elasticsearches
}

// Ingresses returns the list of ingress objects for this strategy. This might be platform-dependent
func (s S) Ingresses() []networkingv1.Ingress {
	return s.ingresses
}

// HorizontalPodAutoscalers returns the list of HPAs objects for this strategy.
func (s S) HorizontalPodAutoscalers() []autoscalingv2beta2.HorizontalPodAutoscaler {
	return s.horizontalPodAutoscalers
}

// Kafkas returns the list of Kafkas for this strategy.
func (s S) Kafkas() []kafkav1beta2.Kafka {
	return s.kafkas
}

// KafkaUsers returns the list of KafkaUsers for this strategy.
func (s S) KafkaUsers() []kafkav1beta2.KafkaUser {
	return s.kafkaUsers
}

// Routes returns the list of routes for this strategy. This might be platform-dependent
func (s S) Routes() []osv1.Route {
	return s.routes
}

// Services returns the list of services for this strategy
func (s S) Services() []corev1.Service {
	return s.services
}

// Secrets returns the list of secrets for this strategy
func (s S) Secrets() []corev1.Secret {
	return s.secrets
}

// Dependencies returns the list of batches for this strategy that are considered dependencies
func (s S) Dependencies() []batchv1.Job {
	return s.dependencies
}

// All returns the list of all objects for this strategy
func (s S) All() []runtime.Object {
	var ret []runtime.Object

	// Keep ordering close to
	// https://github.com/kubernetes-sigs/kustomize/blob/master/api/resid/gvk.go#L77-L103

	for _, o := range s.accounts {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.clusterRoleBindings {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.consoleLinks {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.configMaps {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.cronJobs {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.elasticsearches {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.ingresses {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.horizontalPodAutoscalers {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.kafkas {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.kafkaUsers {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.routes {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.services {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.secrets {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.dependencies {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.daemonSets {
		ret = append(ret, o.DeepCopy())
	}

	for _, o := range s.deployments {
		ret = append(ret, o.DeepCopy())
	}

	return ret
}
