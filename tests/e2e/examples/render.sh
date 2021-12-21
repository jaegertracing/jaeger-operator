#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for examples-agent-as-daemonset test"
cd examples-agent-as-daemonset
export JAEGER_NAME=agent-as-daemonset
export JAEGER_SERVICE=agent-as-daemonset
export JAEGER_OPERATION=smoketestoperation
$GOMPLATE -f $EXAMPLES_DIR/agent-as-daemonset.yaml -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml

cd ..

echo "Rendering templates for examples-business-application-injected-sidecar test"
cd examples-business-application-injected-sidecar
export JAEGER_NAME=simplest
export JAEGER_SERVICE=simplest
export JAEGER_OPERATION=smoketestoperation
cat $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./livenessProbe.yaml > ./00-install.yaml
$GOMPLATE -f $EXAMPLES_DIR/simplest.yaml -o./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o./02-assert.yaml

cd ..

echo "Rendering templates for examples-service-types test"
cd examples-service-types
export JAEGER_SERVICE=service-types
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=service-types
$GOMPLATE -f $EXAMPLES_DIR/service-types.yaml -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml

cd ..

echo "Rendering templates for examples-simple-prod test"
cd examples-simple-prod
export JAEGER_SERVICE=simple-prod
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=simple-prod
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $EXAMPLES_DIR/simple-prod.yaml -o ./01-install.yaml
sed -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml

cd ..

echo "Rendering templates for examples-simple-prod-with-volumes test"
cd examples-simple-prod-with-volumes
export JAEGER_SERVICE=simple-prod-with-volumes
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=simple-prod
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $EXAMPLES_DIR/simple-prod-with-volumes.yaml -o ./01-install.yaml
sed -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml

cd ..

echo "Rendering templates for examples-simplest test"
cd examples-simplest
export JAEGER_SERVICE=smoketest
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=simplest
$GOMPLATE -f $EXAMPLES_DIR/simplest.yaml -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml

cd ..

echo "Rendering templates for examples-with-badger test"
cd examples-with-badger
export JAEGER_SERVICE=with-badger
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=with-badger
$GOMPLATE -f $EXAMPLES_DIR/with-badger.yaml -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml

cd ..


echo "Rendering templates for examples-with-badger-and-volume test"
cd examples-with-badger-and-volume
export JAEGER_SERVICE=with-badger-and-volume
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=with-badger-and-volume
$GOMPLATE -f $EXAMPLES_DIR/with-badger-and-volume.yaml -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml

cd ..

echo "Rendering templates for examples-with-cassandra test"
cd examples-with-cassandra
export JAEGER_SERVICE=with-cassandra
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=with-cassandra
$GOMPLATE -f $TEMPLATES_DIR/cassandra-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/cassandra-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $EXAMPLES_DIR/with-cassandra.yaml -o ./01-install.yaml
sed -i "s~cassandra.default.svc~cassandra~gi" ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml
cd ..

echo "Rendering templates for examples-with-sampling test"
cd examples-with-sampling
export JAEGER_SERVICE=with-sampling
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=with-sampling
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $EXAMPLES_DIR/with-sampling.yaml -o ./01-install.yaml
sed -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml
