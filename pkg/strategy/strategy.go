package strategy

import (
	osv1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"

	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

// S knows what type of deployments to build based on a given spec
type S struct {
	typ                 Type
	accounts            []v1.ServiceAccount
	configMaps          []v1.ConfigMap
	cronJobs            []batchv1beta1.CronJob
	clusterRoleBindings []rbac.ClusterRoleBinding
	daemonSets          []appsv1.DaemonSet
	dependencies        []batchv1.Job
	deployments         []appsv1.Deployment
	elasticsearches     []esv1.Elasticsearch
	ingresses           []v1beta1.Ingress
	routes              []osv1.Route
	services            []v1.Service
	secrets             []v1.Secret
}

// Type represents a specific deployment strategy, like 'all-in-one'
type Type string

const (

	// AllInOne represents the 'all-in-one' deployment strategy
	AllInOne Type = "allInOne"

	// Production represents the 'production' deployment strategy
	Production Type = "production"

	// Streaming represents the 'streaming' deployment strategy
	Streaming Type = "streaming"
)

// New constructs a new strategy from scratch
func New() *S {
	return &S{}
}

// Type returns the strategy type for the given strategy
func (s S) Type() Type {
	return s.typ
}

// WithAccounts returns the strategy with the given list of service accounts
func (s S) WithAccounts(accs []v1.ServiceAccount) S {
	s.accounts = accs
	return s
}

// WithClusterRoleBindings returns the strategy with the given list of config maps
func (s S) WithClusterRoleBindings(c []rbac.ClusterRoleBinding) S {
	s.clusterRoleBindings = c
	return s
}

// WithConfigMaps returns the strategy with the given list of config maps
func (s S) WithConfigMaps(c []v1.ConfigMap) S {
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
func (s S) WithIngresses(i []v1beta1.Ingress) S {
	s.ingresses = i
	return s
}

// WithRoutes returns the strategy with the given list of routes
func (s S) WithRoutes(r []osv1.Route) S {
	s.routes = r
	return s
}

// WithServices returns the strategy with the given list of routes
func (s S) WithServices(svcs []v1.Service) S {
	s.services = svcs
	return s
}

// WithSecrets returns the strategy with the given list of secrets
func (s S) WithSecrets(secrets []v1.Secret) S {
	s.secrets = secrets
	return s
}

// Accounts returns the list of service accounts for this strategy
func (s S) Accounts() []v1.ServiceAccount {
	return s.accounts
}

// ClusterRoleBindings returns the list of cluster role bindings for this strategy
func (s S) ClusterRoleBindings() []rbac.ClusterRoleBinding {
	return s.clusterRoleBindings
}

// ConfigMaps returns the list of config maps for this strategy
func (s S) ConfigMaps() []v1.ConfigMap {
	return s.configMaps
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
func (s S) Ingresses() []v1beta1.Ingress {
	return s.ingresses
}

// Routes returns the list of routes for this strategy. This might be platform-dependent
func (s S) Routes() []osv1.Route {
	return s.routes
}

// Services returns the list of services for this strategy
func (s S) Services() []v1.Service {
	return s.services
}

// Secrets returns the list of secrets for this strategy
func (s S) Secrets() []v1.Secret {
	return s.secrets
}

// Dependencies returns the list of batches for this strategy that are considered dependencies
func (s S) Dependencies() []batchv1.Job {
	return s.dependencies
}
