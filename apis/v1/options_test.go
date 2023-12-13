package v1

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleOption(t *testing.T) {
	o := Options{}
	o.UnmarshalJSON([]byte(`{"key": "value"}`))
	args := o.ToArgs()
	assert.Equal(t, "--key=value", args[0])
}

func TestNoOptions(t *testing.T) {
	o := Options{}
	assert.Empty(t, o.ToArgs())
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
	require.NoError(t, err)
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

func TestUnmarshalToArgs(t *testing.T) {
	tests := []struct {
		in   string
		args []string
		err  string
	}{
		{in: `^`, err: "invalid character '^' looking for beginning of value"},
		{
			in:   `{"a": 5000000000, "b": 15.222, "c":true, "d": "foo"}`,
			args: []string{"--a=5000000000", "--b=15.222", "--c=true", "--d=foo"},
		},
		{
			in:  `{"a": {"b": {"c": [{"d": "e", "f": {"g": {"h": "i"}}}]}}}`,
			err: "invalid option type, expect: string, got: map[string]interface {}",
		},
	}
	for _, test := range tests {
		opts := Options{}
		err := opts.UnmarshalJSON([]byte(test.in))
		if test.err != "" {
			require.EqualError(t, err, test.err)
		} else {
			require.NoError(t, err)
			args := opts.ToArgs()
			sort.SliceStable(args, func(i, j int) bool {
				return args[i] < args[j]
			})
			assert.Equal(t, test.args, args)
		}
	}
}

func TestMultipleSubValuesWithFilter(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"memory": {"max-traces": "50000"}, "es": {"server-urls": "http://elasticsearch:9200", "username": "elastic", "password": "changeme"}}`))
	o = o.Filter("memory")
	args := o.ToArgs()
	assert.Len(t, args, 1)
	assert.Equal(t, "50000", o.Map()["memory.max-traces"])
}

func TestMultipleSubValuesWithFilterWithArchive(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"memory": {"max-traces": "50000"}, "es": {"server-urls": "http://elasticsearch:9200", "username": "elastic", "password": "changeme"}, "es-archive": {"server-urls": "http://elasticsearch2:9200"}}`))
	o = o.Filter("es")
	args := o.ToArgs()
	assert.Len(t, args, 4)
	assert.Equal(t, "http://elasticsearch:9200", o.Map()["es.server-urls"])
	assert.Equal(t, "http://elasticsearch2:9200", o.Map()["es-archive.server-urls"])
	assert.Equal(t, "elastic", o.Map()["es.username"])
	assert.Equal(t, "changeme", o.Map()["es.password"])
}

func TestExposedMap(t *testing.T) {
	o := NewOptions(nil)
	o.UnmarshalJSON([]byte(`{"cassandra": {"servers": "cassandra:9042"}}`))
	assert.Equal(t, "cassandra:9042", o.Map()["cassandra.servers"])
}

func TestMarshallRaw(t *testing.T) {
	json := []byte(`{"cassandra": {"servers": "cassandra:9042"}}`)
	o := NewOptions(nil)
	o.json = &json
	bytes, err := o.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, bytes, json)
}

func TestMarshallEmpty(t *testing.T) {
	o := NewOptions(nil)
	json := []byte(`{}`)
	bytes, err := o.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, bytes, json)
}

func TestUpdate(t *testing.T) {
	// prepare
	o := NewOptions(map[string]interface{}{
		"key": "original",
	})

	// test
	o.Map()["key"] = "new"

	// verify
	assert.Equal(t, "new", o.opts["key"])
}

func TestStringMap(t *testing.T) {
	o := NewOptions(nil)
	err := o.UnmarshalJSON([]byte(`{"firstsarg":"v1", "additional-headers":["whatever:thing", "access-control-allow-origin:blerg"]}`))
	require.NoError(t, err)
	expected := map[string]string{"firstsarg": "v1"}
	strMap := o.StringMap()
	assert.Len(t, strMap, 1)
	assert.Equal(t, expected, strMap)
}

func TestDeepCopy(t *testing.T) {
	o1 := NewOptions(nil)
	err := o1.UnmarshalJSON([]byte(`{"firstsarg":"v1", "additional-headers":["whatever:thing", "access-control-allow-origin:blerg"]}`))
	require.NoError(t, err)
	copy := o1.opts.DeepCopy()

	assert.Equal(t, &(o1.opts), copy)
}

func TestRepetitiveArguments(t *testing.T) {
	o := NewOptions(nil)
	err := o.UnmarshalJSON([]byte(`{"firstsarg":"v1", "additional-headers":["whatever:thing", "access-control-allow-origin:blerg"]}`))
	require.NoError(t, err)
	expected := []string{"--additional-headers=access-control-allow-origin:blerg", "--additional-headers=whatever:thing", "--firstsarg=v1"}

	args := o.ToArgs()
	sort.SliceStable(args, func(i, j int) bool {
		return args[i] < args[j]
	})

	assert.Len(t, args, 3)
	assert.Equal(t, expected, args)
}
