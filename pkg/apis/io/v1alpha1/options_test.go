package v1alpha1

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleOption(t *testing.T) {
	o := Options{}
	o.UnmarshalJSON([]byte(`{"key": "value"}`))
	args := o.ToArgs()
	assert.Equal(t, "--key=value", args[0])
}

func TestNoOptions(t *testing.T) {
	o := Options{}
	assert.Len(t, o.ToArgs(), 0)
}

func TestNestedOption(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"log-level": "debug", "memory": {"max-traces": 10000}}`))
	args := o.ToArgs()
	assert.Len(t, args, 2)

	sort.Strings(args)
	assert.Equal(t, "--log-level=debug", args[0])
	assert.Equal(t, "--memory.max-traces=10000", args[1])
}

func TestMarshalling(t *testing.T) {
	o := NewOptions(map[string]interface{}{
		"es.server-urls": "http://elasticsearch.default.svc:9200",
		"es.username":    "elastic",
		"es.password":    "changeme",
	})

	b, err := json.Marshal(o)
	assert.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"es.password":"changeme"`)
	assert.Contains(t, s, `"es.server-urls":"http://elasticsearch.default.svc:9200"`)
	assert.Contains(t, s, `"es.username":"elastic"`)
}

func TestMarshallingWithFilter(t *testing.T) {
	o := NewOptions(map[string]interface{}{
		"es.server-urls":    "http://elasticsearch.default.svc:9200",
		"memory.max-traces": "50000",
	})
	o = o.Filter("memory")
	args := o.ToArgs()
	assert.Len(t, args, 1)
	assert.Equal(t, "50000", o.Map()["memory.max-traces"])
}

func TestMultipleSubValues(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"es": {"server-urls": "http://elasticsearch:9200", "username": "elastic", "password": "changeme"}}`))
	args := o.ToArgs()
	assert.Len(t, args, 3)
}

func TestMultipleSubValuesWithFilter(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"memory": {"max-traces": "50000"}, "es": {"server-urls": "http://elasticsearch:9200", "username": "elastic", "password": "changeme"}}`))
	o = o.Filter("memory")
	args := o.ToArgs()
	assert.Len(t, args, 1)
	assert.Equal(t, "50000", o.Map()["memory.max-traces"])
}

func TestExposedMap(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"cassandra": {"servers": "cassandra:9042"}}`))
	assert.Equal(t, "cassandra:9042", o.Map()["cassandra.servers"])
}
