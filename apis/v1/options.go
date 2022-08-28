package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Values hold a map, with string as the key and either a string or a slice of strings as the value
type Values map[string]interface{}

// DeepCopy indicate how to do a deep copy of Values type
func (v *Values) DeepCopy() *Values {
	out := make(Values, len(*v))
	for key, val := range *v {
		switch val := val.(type) {
		case string:
			out[key] = val

		case []string:
			out[key] = append([]string(nil), val...)
		}
	}
	return &out
}

// Options defines a common options parameter to the different structs
type Options struct {
	opts Values  `json:"-"`
	json *[]byte `json:"-"`
}

// NewOptions build a new Options object based on the given map
func NewOptions(o map[string]interface{}) Options {
	options := Options{}
	options.parse(o)
	return options
}

// Filter creates a new Options object with just the elements identified by the supplied prefix
func (o *Options) Filter(prefix string) Options {
	options := Options{}
	options.opts = make(map[string]interface{})

	archivePrefix := prefix + "-archive."
	prefix += "."

	for k, v := range o.opts {
		if strings.HasPrefix(k, prefix) || strings.HasPrefix(k, archivePrefix) {
			options.opts[k] = v
		}
	}

	return options
}

// UnmarshalJSON implements an alternative parser for this field
func (o *Options) UnmarshalJSON(b []byte) error {
	var entries map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := d.Decode(&entries); err != nil {
		return err
	}
	if err := o.parse(entries); err != nil {
		return err
	}
	o.json = &b
	return nil
}

// MarshalJSON specifies how to convert this object into JSON
func (o Options) MarshalJSON() ([]byte, error) {
	if nil != o.json {
		return *o.json, nil
	}

	if len(o.opts) == 0 {
		return []byte("{}"), nil
	}

	if len(o.opts) > 0 {
		return json.Marshal(o.opts)
	}

	return *o.json, nil
}

func (o *Options) parse(entries map[string]interface{}) error {
	o.json = nil
	o.opts = make(map[string]interface{})
	var err error
	for k, v := range entries {
		o.opts, err = entry(o.opts, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func entry(entries map[string]interface{}, key string, value interface{}) (map[string]interface{}, error) {
	switch val := value.(type) {
	case map[string]interface{}:
		var err error
		for k, v := range val {
			entries, err = entry(entries, fmt.Sprintf("%s.%v", key, k), v)
			if err != nil {
				return nil, err
			}
		}
	case []interface{}: // NOTE: content of the argument list is not returned as []string when decoding json.
		values := make([]string, 0, len(val))
		for _, v := range val {
			str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid option type, expect: string, got: %T", v)
			}
			values = append(values, str)
		}
		entries[key] = values
	case interface{}:
		entries[key] = fmt.Sprintf("%v", value)
	}

	return entries, nil
}

// ToArgs converts the options to a value suitable for the Container.Args field
func (o *Options) ToArgs() []string {
	if len(o.opts) > 0 {
		args := make([]string, 0, len(o.opts))
		for k, v := range o.opts {
			switch v := v.(type) {
			case string:
				args = append(args, fmt.Sprintf("--%s=%v", k, v))
			case []string:
				for _, vv := range v {
					args = append(args, fmt.Sprintf("--%s=%v", k, vv))
				}
			}
		}
		return args
	}
	return nil
}

// Map returns a map representing the option entries. Items are flattened, with dots as separators. For instance
// an option "cassandra" with a nested "servers" object becomes an entry with the key "cassandra.servers"
func (o *Options) Map() map[string]interface{} {
	return o.opts
}

// StringMap returns a map representing the option entries,excluding entries that have multiple values.
// Items are flattened, with dots as separators in the same way as Map does.
func (o *Options) StringMap() map[string]string {
	smap := make(map[string]string)
	for k, v := range o.opts {
		switch v := v.(type) {
		case string:
			smap[k] = v
		}
	}
	return smap
}

// GenericMap returns the map representing the option entries as interface{}, suitable for usage with NewOptions()
func (o *Options) GenericMap() map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range o.opts {
		out[k] = v
	}
	return out
}
