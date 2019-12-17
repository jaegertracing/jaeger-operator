Changes by Version
==================

1.16.0 (2019-12-17)
-------------------

Breaking changes:
* None

Other noteworthy changes:
* Fixed permissions for ServiceMonitor objects ([#831](https://github.com/jaegertracing/jaeger-operator/pull/831))
* Add timeout for Cassandra Schema creation job ([#820](https://github.com/jaegertracing/jaeger-operator/pull/820))
* Fixed the with-badger-and-volume example ([#827](https://github.com/jaegertracing/jaeger-operator/pull/827))
* Run rollover cronjob by default daily at midnight ([#812](https://github.com/jaegertracing/jaeger-operator/pull/812))
* Added basic status to CR{D} ([#802](https://github.com/jaegertracing/jaeger-operator/pull/802))
* Disabled tracing by default ([#805](https://github.com/jaegertracing/jaeger-operator/pull/805))
* Remove unnecessary options from auto-kafka-prov example ([#810](https://github.com/jaegertracing/jaeger-operator/pull/810))
* Use APIReader for Get/List resources on the autodetect functions ([#814](https://github.com/jaegertracing/jaeger-operator/pull/814))
* Updated Operator SDK to v0.12.0 ([#799](https://github.com/jaegertracing/jaeger-operator/pull/799))
* Added OpenTelemetry instrumentation ([#738](https://github.com/jaegertracing/jaeger-operator/pull/738))
* Fixed nil pointer when no Jaeger is suitable for sidecar injection ([#783](https://github.com/jaegertracing/jaeger-operator/pull/783))
* CSV changes to be picked up for next release ([#772](https://github.com/jaegertracing/jaeger-operator/pull/772))
* Correctly expose UDP container ports of injected sidecar containers ([#773](https://github.com/jaegertracing/jaeger-operator/pull/773))
* Scan deployments for agent injection ([#454](https://github.com/jaegertracing/jaeger-operator/pull/454))

1.15.0 (2019-11-09)
-------------------

Breaking changes:
* Breaking change - removed legacy io.jaegertracing CRD ([#649](https://github.com/jaegertracing/jaeger-operator/pull/649))

Other noteworthy changes:
* fix sampling strategy file issue in Jaeger Collector ([#741](https://github.com/jaegertracing/jaeger-operator/pull/741))
* Enable tag/digest to be specified in the image parameters to the operator ([#743](https://github.com/jaegertracing/jaeger-operator/pull/743))
* Upgrade deprecated flags from 1.14 and previous, to 1.15 ([#730](https://github.com/jaegertracing/jaeger-operator/pull/730))
* Use StatefulSet from apps/v1 API for ES and Cassandra ([#727](https://github.com/jaegertracing/jaeger-operator/pull/727))
* Read the service account's namespace when POD_NAMESPACE is missing ([#722](https://github.com/jaegertracing/jaeger-operator/pull/722))
* Added automatic provisioning of Kafka when its operator is available ([#713](https://github.com/jaegertracing/jaeger-operator/pull/713))
* New DeploymentStrategy type for JaegerSpec.Strategy  ([#704](https://github.com/jaegertracing/jaeger-operator/pull/704))
* Added workflows publishing the 'master' container image ([#718](https://github.com/jaegertracing/jaeger-operator/pull/718))
* Added labels to cronjob pod template ([#707](https://github.com/jaegertracing/jaeger-operator/pull/707))
* Pass only specified options to spark dependencies ([#708](https://github.com/jaegertracing/jaeger-operator/pull/708))
* Updated Operator SDK to v0.11.0 ([#695](https://github.com/jaegertracing/jaeger-operator/pull/695))
* Update gopkg.in/yaml.v2 dependency to v2.2.4 ([#699](https://github.com/jaegertracing/jaeger-operator/pull/699))
* added cassandra creds ([#590](https://github.com/jaegertracing/jaeger-operator/pull/590))
* Updated the business-application example ([#693](https://github.com/jaegertracing/jaeger-operator/pull/693))
* Add support for TLS on ingress ([#681](https://github.com/jaegertracing/jaeger-operator/pull/681))
* Add support to SuccessfulJobsHistoryLimit ([#621](https://github.com/jaegertracing/jaeger-operator/pull/621))
* Add prometheus annotations to sidecar's deployment ([#684](https://github.com/jaegertracing/jaeger-operator/pull/684))
* add missing grpc port ([#680](https://github.com/jaegertracing/jaeger-operator/pull/680))
* Recognize when a resource has been deleted while the operator waits ([#672](https://github.com/jaegertracing/jaeger-operator/pull/672))
* Enable the documentation URL in the default menu items to be configured via the operator CLI ([#666](https://github.com/jaegertracing/jaeger-operator/pull/666))
* Adjusted the ALM examples and operator capabilities in CSV ([#665](https://github.com/jaegertracing/jaeger-operator/pull/665))
* Bring jaeger operator repo inline with contributing guidelines in mai… ([#664](https://github.com/jaegertracing/jaeger-operator/pull/664))
* Fix error handling when getting environment variable value ([#661](https://github.com/jaegertracing/jaeger-operator/pull/661))
* Update install-sdk to work on Mac ([#660](https://github.com/jaegertracing/jaeger-operator/pull/660))
* Improved the install-sdk target ([#653](https://github.com/jaegertracing/jaeger-operator/pull/653))
* Use elasticsearch operator 4.2, add workflow for 4.1 ([#631](https://github.com/jaegertracing/jaeger-operator/pull/631))
* Load env variables in the given secretName in Spark dependencies ([#651](https://github.com/jaegertracing/jaeger-operator/pull/651))
* Added default agent tags ([#648](https://github.com/jaegertracing/jaeger-operator/pull/648))

1.14.0 (2019-09-04)
-------------------

* Add commonSpec to other jobs (es-index-cleaner, es-rollover, cassandr… ([#640](https://github.com/jaegertracing/jaeger-operator/pull/640))
* Add common spec to dependencies ([#637](https://github.com/jaegertracing/jaeger-operator/pull/637))
* Add resource limits for spark dependencies cronjob ([#620](https://github.com/jaegertracing/jaeger-operator/pull/620))
* Add Jaeger version to Elasticsearch job images ([#628](https://github.com/jaegertracing/jaeger-operator/pull/628))
* Add badger to supported list of storage types ([#616](https://github.com/jaegertracing/jaeger-operator/pull/616))
* Get rid of finalizer, clean sidecars when no jaeger instance found ([#575](https://github.com/jaegertracing/jaeger-operator/pull/575))
* Deploy production ready self provisioned ES by default ([#585](https://github.com/jaegertracing/jaeger-operator/pull/585))
* Always deploy client,data nodes with master node ([#586](https://github.com/jaegertracing/jaeger-operator/pull/586))
* Configure index cleaner properly when rollover is enabled ([#587](https://github.com/jaegertracing/jaeger-operator/pull/587))
* Agent service ports with correct protocol ([#579](https://github.com/jaegertracing/jaeger-operator/pull/579))
* Renamed the ManagedBy label to OperatedBy ([#576](https://github.com/jaegertracing/jaeger-operator/pull/576))
* Added htpasswd option to the OpenShift OAuth type ([#573](https://github.com/jaegertracing/jaeger-operator/pull/573))
* Changed Operator to set ownership of the instances it manages ([#571](https://github.com/jaegertracing/jaeger-operator/pull/571))
* Added upgrade mechanism for managed Jaeger instances ([#476](https://github.com/jaegertracing/jaeger-operator/pull/476))
* Check and update finalizers before setting APIVersion and Kind ([#558](https://github.com/jaegertracing/jaeger-operator/pull/558))
* Remove sidecar when instance is deleted ([#453](https://github.com/jaegertracing/jaeger-operator/pull/453))
* Allow setting es-operator-image ([#549](https://github.com/jaegertracing/jaeger-operator/pull/549))
* Use zero redundancy when number of ES nodes is 1 ([#539](https://github.com/jaegertracing/jaeger-operator/pull/539))
* Use es-operator from 4.1 branch ([#537](https://github.com/jaegertracing/jaeger-operator/pull/537))
* Reinstated the service metrics ([#530](https://github.com/jaegertracing/jaeger-operator/pull/530))
* Use ES single redundancy by default ([#531](https://github.com/jaegertracing/jaeger-operator/pull/531))
* Change replace method, to remain compatible with golang 1.11 ([#529](https://github.com/jaegertracing/jaeger-operator/pull/529))
* Avoid touching the original structure of the options. ([#523](https://github.com/jaegertracing/jaeger-operator/pull/523))
* Prevented the Operator from overriding Secrets/ImagePullSecrets on ServiceAccounts ([#526](https://github.com/jaegertracing/jaeger-operator/pull/526))
* Added support for OpenShift-specific OAuth Proxy options ([#508](https://github.com/jaegertracing/jaeger-operator/pull/508))
* Allowed usage of custom SA for OAuth Proxy ([#520](https://github.com/jaegertracing/jaeger-operator/pull/520))
* Make sure the ES operator's UUID is a valid DNS name ([#515](https://github.com/jaegertracing/jaeger-operator/pull/515))
* Set the ES node GenUUID to explicit value based on jaeger instance namespace and name ([#495](https://github.com/jaegertracing/jaeger-operator/pull/495))
* Add linkerd.io/inject=disabled annotation ([#507](https://github.com/jaegertracing/jaeger-operator/pull/507))


1.13.1 (2019-07-05)
-------------------

* Bump Jaeger to 1.13 ([#504](https://github.com/jaegertracing/jaeger-operator/pull/504))
* Disable the property ttlSecondsAfterFinished ([#503](https://github.com/jaegertracing/jaeger-operator/pull/503))
* Set default redundancy policy to zero ([#501](https://github.com/jaegertracing/jaeger-operator/pull/501))

1.13.0 (2019-07-02)
-------------------

* Changed to always use namespace when a name is involved ([#485](https://github.com/jaegertracing/jaeger-operator/pull/485))
* Sanitize names that must follow DNS naming rules ([#483](https://github.com/jaegertracing/jaeger-operator/pull/483))
* Added instructions for daemonsets on OpenShift ([#346](https://github.com/jaegertracing/jaeger-operator/pull/346))
* Enable completion time-to-live to be set on all jobs ([#407](https://github.com/jaegertracing/jaeger-operator/pull/407))

1.12.1 (2019-06-06)
-------------------

* Removed 'expose metrics port' to prevent 'failed to create or get service' error ([#462](https://github.com/jaegertracing/jaeger-operator/pull/462))
* Add support for securityContext and serviceAccount ([#456](https://github.com/jaegertracing/jaeger-operator/pull/456))
* Add install SDK goal to make ([#458](https://github.com/jaegertracing/jaeger-operator/pull/458))
* Upgraded the operator-sdk version to 0.8.1 ([#449](https://github.com/jaegertracing/jaeger-operator/pull/449))
* Switch to go modules from dep ([#449](https://github.com/jaegertracing/jaeger-operator/pull/449))
* Do not set a default Elasticsearch image ([#450](https://github.com/jaegertracing/jaeger-operator/pull/450))
* Log the operator image name when created ([#452](https://github.com/jaegertracing/jaeger-operator/pull/452))
* Add label to the common spec ([#445](https://github.com/jaegertracing/jaeger-operator/pull/445))
* Fix injecting volumes into rollover jobs ([#446](https://github.com/jaegertracing/jaeger-operator/pull/446))
* Remove race condition by disabling esIndexCleaner till after SmokeTes… ([#437](https://github.com/jaegertracing/jaeger-operator/pull/437))
* Fix runtime panic when trying to update operator controlled resources that don't have annotation or labels   ([#433](https://github.com/jaegertracing/jaeger-operator/pull/433))

1.12.0 (2019-05-22)
-------------------

* Update to 1.12 and use new admin ports ([#425](https://github.com/jaegertracing/jaeger-operator/pull/425))
* Use ephemeral storage for Kafka tests ([#419](https://github.com/jaegertracing/jaeger-operator/pull/419))
* Fix csv example and add spec.maturity ([#416](https://github.com/jaegertracing/jaeger-operator/pull/416))
* Add resources requests/limits to oauth_proxy ([#410](https://github.com/jaegertracing/jaeger-operator/pull/410))
* Check that context is not nil before calling cleanup ([#413](https://github.com/jaegertracing/jaeger-operator/pull/413))
* Improve error message when queries fail ([#402](https://github.com/jaegertracing/jaeger-operator/pull/402))
* Add resource requirements to sidecar agent ([#401](https://github.com/jaegertracing/jaeger-operator/pull/401))
* Add streaming e2e tests ([#400](https://github.com/jaegertracing/jaeger-operator/pull/400))
* Make sure to call ctx.cleanup if perpare()) fails ([#389](https://github.com/jaegertracing/jaeger-operator/pull/389))
* Change how Kafka is configured for collector and ingester ([#390](https://github.com/jaegertracing/jaeger-operator/pull/390))
* Use storage namespace in index cleaner test ([#382](https://github.com/jaegertracing/jaeger-operator/pull/382))
* Fix rbac policy issue with blockOwnerDeletion ([#384](https://github.com/jaegertracing/jaeger-operator/pull/384))
* Reinstate gosec with fix for OOM error ([#381](https://github.com/jaegertracing/jaeger-operator/pull/381))
* Enhance ES index cleaner e2e test to verify indices have been removed ([#378](https://github.com/jaegertracing/jaeger-operator/pull/378))
* Add owner ref on operator's service to ensure it gets deleted when op… ([#377](https://github.com/jaegertracing/jaeger-operator/pull/377))
* Update CSV description to comply with guidelines ([#374](https://github.com/jaegertracing/jaeger-operator/pull/374))
* Include elasticsearch statefulset nodes in availability check ([#371](https://github.com/jaegertracing/jaeger-operator/pull/371))
* Fail lint goal if not empty ([#372](https://github.com/jaegertracing/jaeger-operator/pull/372))

1.11.1 (2019-04-09)
-------------------

* Include docs for common config ([#367](https://github.com/jaegertracing/jaeger-operator/pull/367))
* Reinstated the registration of ES types ([#366](https://github.com/jaegertracing/jaeger-operator/pull/366))
* Add support for affinity and tolerations ([#361](https://github.com/jaegertracing/jaeger-operator/pull/361))
* Support injection of JAEGER_SERVICE_NAME based on app or k8s recommended labels ([#362](https://github.com/jaegertracing/jaeger-operator/pull/362))
* Change ES operator apiversion ([#360](https://github.com/jaegertracing/jaeger-operator/pull/360))
* Update test to run on OpenShift ([#350](https://github.com/jaegertracing/jaeger-operator/pull/350))
* Add prometheus scrape 'false' annotation to headless collector service ([#348](https://github.com/jaegertracing/jaeger-operator/pull/348))
* Derive agent container/host ports from options if specified ([#353](https://github.com/jaegertracing/jaeger-operator/pull/353))

1.11.0 (2019-03-22)
-------------------

### Breaking changes

* Moved from v1alpha1 to v1 ([#265](https://github.com/jaegertracing/jaeger-operator/pull/265))
* Use storage flags instead of CR properties for spark job ([#295](https://github.com/jaegertracing/jaeger-operator/pull/295))
* Changed from 'size' to 'replicas' ([#271](https://github.com/jaegertracing/jaeger-operator/pull/271)). "Size" will still work for the next couple of releases.

### Other changes

* Initialise menu to include Log Out option when using OAuth Proxy ([#344](https://github.com/jaegertracing/jaeger-operator/pull/344))
* Change Operator provider to CNCF ([#263](https://github.com/jaegertracing/jaeger-operator/pull/263))
* Added note about the apiVersion used up to 1.10.0 ([#283](https://github.com/jaegertracing/jaeger-operator/pull/283))
* Implemented a second service for the collector ([#339](https://github.com/jaegertracing/jaeger-operator/pull/339))
* Enabled DNS as the service discovery mechanism for agent => collector communication ([#333](https://github.com/jaegertracing/jaeger-operator/pull/333))
* Sorted the container arguments inside deployments ([#337](https://github.com/jaegertracing/jaeger-operator/pull/337))
* Use client certs for elasticsearch ([#325](https://github.com/jaegertracing/jaeger-operator/pull/325))
* Load back Elasticsearch certs from secrets ([#324](https://github.com/jaegertracing/jaeger-operator/pull/324))
* Disable spark dependencies for self provisioned es ([#319](https://github.com/jaegertracing/jaeger-operator/pull/319))
* Remove index cleaner from prod-es-deploy example ([#314](https://github.com/jaegertracing/jaeger-operator/pull/314))
* Set default query timeout for provisioned ES ([#313](https://github.com/jaegertracing/jaeger-operator/pull/313))
* Automatically Enable/disable depenencies tab ([#311](https://github.com/jaegertracing/jaeger-operator/pull/311))
* Unmarshall numbers in options to number not float64 ([#308](https://github.com/jaegertracing/jaeger-operator/pull/308))
* Inject archive index configuration for provisioned ES ([#309](https://github.com/jaegertracing/jaeger-operator/pull/309))
* update #305, add grps and health port to jaeger collector service ([#306](https://github.com/jaegertracing/jaeger-operator/pull/306))
* Enable archive button if archive storage is enabled ([#303](https://github.com/jaegertracing/jaeger-operator/pull/303))
* Fix reverting ingress security to oauth-proxy on openshift if set to none ([#301](https://github.com/jaegertracing/jaeger-operator/pull/301))
* Change agent reporter to GRPC ([#299](https://github.com/jaegertracing/jaeger-operator/pull/299))
* Bump jaeger version to 1.11 ([#300](https://github.com/jaegertracing/jaeger-operator/pull/300))
* Enable agent readiness probe ([#297](https://github.com/jaegertracing/jaeger-operator/pull/297))
* Use storage flags instead of CR properties for spark job ([#295](https://github.com/jaegertracing/jaeger-operator/pull/295))
* Change operator.yaml to use master, to keep the readme uptodate with latest version ([#296](https://github.com/jaegertracing/jaeger-operator/pull/296))
* Add Elasticsearch image to CR and flag ([#289](https://github.com/jaegertracing/jaeger-operator/pull/289))
* Updated to Operator SDK 0.5.0 ([#273](https://github.com/jaegertracing/jaeger-operator/pull/273))
* Block until objects have been created and are ready ([#279](https://github.com/jaegertracing/jaeger-operator/pull/279))
* Add rollover support ([#267](https://github.com/jaegertracing/jaeger-operator/pull/267))
* Added publishing of major.minor image for the operator ([#274](https://github.com/jaegertracing/jaeger-operator/pull/274))
* Use only ES data nodes to calculate shards ([#257](https://github.com/jaegertracing/jaeger-operator/pull/257))
* Reinstated sidecar for query, plus small refactoring of sidecar ([#246](https://github.com/jaegertracing/jaeger-operator/pull/246))
* Remove ES master certs ([#256](https://github.com/jaegertracing/jaeger-operator/pull/256))
* Store back the CR only if it has changed ([#249](https://github.com/jaegertracing/jaeger-operator/pull/249))
* Fixed role rule for Elasticsearch ([#251](https://github.com/jaegertracing/jaeger-operator/pull/251))
* Wait for elasticsearch cluster to be up ([#242](https://github.com/jaegertracing/jaeger-operator/pull/242))

1.10.0 (2019-02-28)
-------------------

* Automatically detect when the ES operator is available ([#239](https://github.com/jaegertracing/jaeger-operator/pull/239))
* Adjusted logs to be consistent across the code base ([#237](https://github.com/jaegertracing/jaeger-operator/pull/237))
* Fixed deployment of Elasticsearch via its operator ([#234](https://github.com/jaegertracing/jaeger-operator/pull/234))
* Set ES shards and replicas based on redundancy policy ([#229](https://github.com/jaegertracing/jaeger-operator/pull/229))
* Update Jaeger CR ([#193](https://github.com/jaegertracing/jaeger-operator/pull/193))
* Add storage secrets to es-index-cleaner cronjob ([#197](https://github.com/jaegertracing/jaeger-operator/pull/197))
* Removed constraint on namespace when obtaining available Jaeger instances ([#213](https://github.com/jaegertracing/jaeger-operator/pull/213))
* Added workaround for kubectl logs and get pods commands ([#225](https://github.com/jaegertracing/jaeger-operator/pull/225))
* Add -n observability so kubectl get deployment command works correctly ([#223](https://github.com/jaegertracing/jaeger-operator/pull/223))
* Added capability of detecting the platform ([#217](https://github.com/jaegertracing/jaeger-operator/pull/217))
* Deploy one ES node ([#221](https://github.com/jaegertracing/jaeger-operator/pull/221))
* Use centos image ([#220](https://github.com/jaegertracing/jaeger-operator/pull/220))
* Add support for deploying elasticsearch ([#191](https://github.com/jaegertracing/jaeger-operator/pull/191))
* Replaced use of strings.ToLower comparison with EqualFold ([#214](https://github.com/jaegertracing/jaeger-operator/pull/214))
* Bump Jaeger to 1.10 ([#212](https://github.com/jaegertracing/jaeger-operator/pull/212))
* Ignore golang coverage html ([#208](https://github.com/jaegertracing/jaeger-operator/pull/208))

1.9.2 (2019-02-11)
------------------

* Enable single operator to monitor all namespaces ([#188](https://github.com/jaegertracing/jaeger-operator/pull/188))
* Added flag to control the logging level ([#202](https://github.com/jaegertracing/jaeger-operator/pull/202))
* Updated operator-sdk to v0.4.1 ([#200](https://github.com/jaegertracing/jaeger-operator/pull/200))
* Added newline to the end of the role YAML file ([#199](https://github.com/jaegertracing/jaeger-operator/pull/199))
* Added mention to WATCH_NAMESPACE when running for OpenShift ([#195](https://github.com/jaegertracing/jaeger-operator/pull/195))
* Added openshift route to role ([#198](https://github.com/jaegertracing/jaeger-operator/pull/198))
* Added Route to SDK Scheme ([#194](https://github.com/jaegertracing/jaeger-operator/pull/194))
* Add Jaeger CSV and Package for OLM integration and deployment of the … ([#173](https://github.com/jaegertracing/jaeger-operator/pull/173))

1.9.1 (2019-01-30)
------------------

* Remove debug logging from simple-streaming example ([#185](https://github.com/jaegertracing/jaeger-operator/pull/185))
* Add ingester (and kafka) support ([#168](https://github.com/jaegertracing/jaeger-operator/pull/168))
* When filtering storage options, also include '-archive' related options ([#182](https://github.com/jaegertracing/jaeger-operator/pull/182))

1.9.0 (2019-01-23)
------------------

* Changed to use recommended labels ([#172](https://github.com/jaegertracing/jaeger-operator/pull/172))
* Enable dependencies and index cleaner by default ([#162](https://github.com/jaegertracing/jaeger-operator/pull/162))
* Fix log when spak depenencies are used with unsupported storage ([#161](https://github.com/jaegertracing/jaeger-operator/pull/161))
* Fix serviceaccount could not be created by the operator on openshift. ([#165](https://github.com/jaegertracing/jaeger-operator/pull/165))
* Add Elasticsearch index cleaner as cron job ([#155](https://github.com/jaegertracing/jaeger-operator/pull/155))
* Fix import order for collector-test ([#158](https://github.com/jaegertracing/jaeger-operator/pull/158))
* Smoke test ([#145](https://github.com/jaegertracing/jaeger-operator/pull/145))
* Add deploy clean target and rename es/cass to deploy- ([#149](https://github.com/jaegertracing/jaeger-operator/pull/149))
* Add spark job ([#140](https://github.com/jaegertracing/jaeger-operator/pull/140))
* Automatically format imports ([#151](https://github.com/jaegertracing/jaeger-operator/pull/151))
* Silence 'mkdir' from e2e-tests ([#153](https://github.com/jaegertracing/jaeger-operator/pull/153))
* Move pkg/configmap to pkg/config/ui ([#152](https://github.com/jaegertracing/jaeger-operator/pull/152))
* Fix secrets readme ([#150](https://github.com/jaegertracing/jaeger-operator/pull/150))

1.8.2 (2018-12-03)
------------------

* Configure sampling strategies ([#139](https://github.com/jaegertracing/jaeger-operator/pull/139))
* Add support for secrets ([#114](https://github.com/jaegertracing/jaeger-operator/pull/114))
* Fix crd links ([#132](https://github.com/jaegertracing/jaeger-operator/pull/132))
* Create e2e testdir, fix contributing readme ([#131](https://github.com/jaegertracing/jaeger-operator/pull/131))
* Enable JAEGER_SERVICE_NAME and JAEGER_PROPAGATION env vars to be set … ([#128](https://github.com/jaegertracing/jaeger-operator/pull/128))
* Add CRD to install steps, and update cleanup instructions ([#129](https://github.com/jaegertracing/jaeger-operator/pull/129))
* Rename controller to strategy ([#125](https://github.com/jaegertracing/jaeger-operator/pull/125))
* Add tests for new operator-sdk related code ([#122](https://github.com/jaegertracing/jaeger-operator/pull/122))
* Update README.adoc to match yaml files in deploy ([#124](https://github.com/jaegertracing/jaeger-operator/pull/124))

1.8.1 (2018-11-21)
------------------

* Add support for UI configuration ([#115](https://github.com/jaegertracing/jaeger-operator/pull/115))
* Use proper jaeger-operator version for e2e tests and remove readiness check from DaemonSet ([#120](https://github.com/jaegertracing/jaeger-operator/pull/120))
* Migrate to Operator SDK 0.1.0 ([#116](https://github.com/jaegertracing/jaeger-operator/pull/116))
* Fix changelog 'new features' header for 1.8 ([#113](https://github.com/jaegertracing/jaeger-operator/pull/113))

1.8.0 (2018-11-13)
------------------

*Notable new Features*

* Query base path should be used to configure correct path in ingress ([#108](https://github.com/jaegertracing/jaeger-operator/pull/108))
* Enable resources to be defined at top level and overridden at compone… ([#110](https://github.com/jaegertracing/jaeger-operator/pull/110))
* Add OAuth Proxy to UI when on OpenShift ([#100](https://github.com/jaegertracing/jaeger-operator/pull/100))
* Enable top level annotations to be defined ([#97](https://github.com/jaegertracing/jaeger-operator/pull/97))
* Support volumes and volumeMounts ([#82](https://github.com/jaegertracing/jaeger-operator/pull/82))
* Add support for OpenShift routes ([#93](https://github.com/jaegertracing/jaeger-operator/pull/93))
* Enable annotations to be specified with the deployable components ([#86](https://github.com/jaegertracing/jaeger-operator/pull/86))
* Add support for Cassandra create-schema job ([#71](https://github.com/jaegertracing/jaeger-operator/pull/71))
* Inject sidecar in properly annotated pods ([#58](https://github.com/jaegertracing/jaeger-operator/pull/58))
* Support deployment of agent as a DaemonSet ([#52](https://github.com/jaegertracing/jaeger-operator/pull/52))

*Breaking changes*

* Change CRD to use lower camel case ([#87](https://github.com/jaegertracing/jaeger-operator/pull/87))
* Factor out ingress from all-in-one and query, as common to both but i… ([#91](https://github.com/jaegertracing/jaeger-operator/pull/91))
* Remove zipkin service ([#75](https://github.com/jaegertracing/jaeger-operator/pull/75))

*Full list of commits:*

* Query base path should be used to configure correct path in ingress ([#108](https://github.com/jaegertracing/jaeger-operator/pull/108))
* Enable resources to be defined at top level and overridden at compone… ([#110](https://github.com/jaegertracing/jaeger-operator/pull/110))
* Fix disable-oauth-proxy example ([#107](https://github.com/jaegertracing/jaeger-operator/pull/107))
* Add OAuth Proxy to UI when on OpenShift ([#100](https://github.com/jaegertracing/jaeger-operator/pull/100))
* Refactor common spec elements into a single struct with common proces… ([#105](https://github.com/jaegertracing/jaeger-operator/pull/105))
* Ensure 'make generate' has been executed when model changes are made ([#101](https://github.com/jaegertracing/jaeger-operator/pull/101))
* Enable top level annotations to be defined ([#97](https://github.com/jaegertracing/jaeger-operator/pull/97))
* Update generated code and reverted change to 'all-in-one' in CRD ([#98](https://github.com/jaegertracing/jaeger-operator/pull/98))
* Support volumes and volumeMounts ([#82](https://github.com/jaegertracing/jaeger-operator/pull/82))
* Update readme to include info about storage options being located in … ([#96](https://github.com/jaegertracing/jaeger-operator/pull/96))
* Enable storage options to be filtered out based on specified storage … ([#94](https://github.com/jaegertracing/jaeger-operator/pull/94))
* Add support for OpenShift routes ([#93](https://github.com/jaegertracing/jaeger-operator/pull/93))
* Change CRD to use lower camel case ([#87](https://github.com/jaegertracing/jaeger-operator/pull/87))
* Factor out ingress from all-in-one and query, as common to both but i… ([#91](https://github.com/jaegertracing/jaeger-operator/pull/91))
* Fix operator SDK version as master is too unpredicatable at the moment ([#92](https://github.com/jaegertracing/jaeger-operator/pull/92))
* Update generated file after new annotations field ([#90](https://github.com/jaegertracing/jaeger-operator/pull/90))
* Enable annotations to be specified with the deployable components ([#86](https://github.com/jaegertracing/jaeger-operator/pull/86))
* Remove zipkin service ([#75](https://github.com/jaegertracing/jaeger-operator/pull/75))
* Add support for Cassandra create-schema job ([#71](https://github.com/jaegertracing/jaeger-operator/pull/71))
* Fix table of contents on readme ([#73](https://github.com/jaegertracing/jaeger-operator/pull/73))
* Update the Operator SDK version ([#69](https://github.com/jaegertracing/jaeger-operator/pull/69))
* Add sidecar.istio.io/inject=false annotation to all-in-one, agent (da… ([#67](https://github.com/jaegertracing/jaeger-operator/pull/67))
* Fix zipkin port issue ([#65](https://github.com/jaegertracing/jaeger-operator/pull/65))
* Go 1.11.1 ([#61](https://github.com/jaegertracing/jaeger-operator/pull/61))
* Inject sidecar in properly annotated pods ([#58](https://github.com/jaegertracing/jaeger-operator/pull/58))
* Support deployment of agent as a DaemonSet ([#52](https://github.com/jaegertracing/jaeger-operator/pull/52))
* Normalize options on the stub and update the normalized CR ([#54](https://github.com/jaegertracing/jaeger-operator/pull/54))
* Document the disable ingress feature ([#55](https://github.com/jaegertracing/jaeger-operator/pull/55))
* dep ensure ([#51](https://github.com/jaegertracing/jaeger-operator/pull/51))
* Add support for JaegerIngressSpec to all-in-one

1.7.0 (2018-09-25)
------------------

This release brings Jaeger v1.7 to the Operator.

*Full list of commits:*

* Release v1.7.0
* Bump Jaeger to 1.7 ([#41](https://github.com/jaegertracing/jaeger-operator/pull/41))

1.6.5 (2018-09-21)
------------------

This is our initial release based on Jaeger 1.6.

*Full list of commits:*

* Release v1.6.5
* Push the tag with the new commit to master, not the release tag
* Fix git push syntax
* Push tag to master
* Merge release commit into master ([#39](https://github.com/jaegertracing/jaeger-operator/pull/39))
* Add query ingress enable switch ([#36](https://github.com/jaegertracing/jaeger-operator/pull/36))
* Fix the run goal ([#35](https://github.com/jaegertracing/jaeger-operator/pull/35))
* Release v1.6.1
* Add 'build' step when publishing image
* Fix docker push command and update release instructions
* Add release scripts ([#32](https://github.com/jaegertracing/jaeger-operator/pull/32))
* Fix command to deploy the simplest operator ([#34](https://github.com/jaegertracing/jaeger-operator/pull/34))
* Add IntelliJ specific files to gitignore ([#33](https://github.com/jaegertracing/jaeger-operator/pull/33))
* Add prometheus scrape annotations to Jaeger collector, query and all-in-one ([#27](https://github.com/jaegertracing/jaeger-operator/pull/27))
* Remove work in progress notice
* Add instructions on how to run the operator on OpenShift
* Support Jaeger version and image override
* Fix publishing of release
* Release Docker image upon merge to master
* Reuse the same ES for all tests
* Improved how to execute the e2e tests
* Correct uninstall doc to reference delete not create ([#16](https://github.com/jaegertracing/jaeger-operator/pull/16))
* Set ENTRYPOINT for Dockerfile
* Run 'docker' target only before e2e-tests
* 'dep ensure' after adding Cobra/Viper
* Update the Jaeger Operator version at build time
* Add ingress permission to the jaeger-operator
* Install golint/gosec
* Disabled e2e tests on Travis
* Initial working version
* INITIAL COMMIT
