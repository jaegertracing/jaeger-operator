package strategy

import (
	osv1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

// S knows what type of deployments to build based on a given spec
type S struct {
	typ          Type
	accounts     []v1.ServiceAccount
	configMaps   []v1.ConfigMap
	cronJobs     []batchv1beta1.CronJob
	daemonSets   []appsv1.DaemonSet
	dependencies []batchv1.Job
	deployments  []appsv1.Deployment
	ingresses    []v1beta1.Ingress
	routes       []osv1.Route
	services     []v1.Service
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

// WithDeployments returns the strategy with the given list of deployments
func (s S) WithDeployments(deps []appsv1.Deployment) S {
	s.deployments = deps
	return s
}

// WithDependencies returns the strategy with the given list of dependencies
func (s S) WithDependencies(deps []batchv1.Job) S {
	s.dependencies = deps
	return s
}

// Accounts returns the list of service accounts for this strategy
func (s S) Accounts() []v1.ServiceAccount {
	return s.accounts
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

// Dependencies returns the list of batches for this strategy that are considered dependencies
func (s S) Dependencies() []batchv1.Job {
	return s.dependencies
}
