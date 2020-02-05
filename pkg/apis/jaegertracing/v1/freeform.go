package v1

import (
	"encoding/json"
)

// FreeForm defines a common options parameter that maintains the hierarchical structure of the data, unlike Options which flattens the hierarchy into a key/value map where the hierarchy is converted to '.' separated items in the key.
type FreeForm struct {
	json *[]byte `json:"-"`
}

// NewFreeForm build a new FreeForm object based on the given map
func NewFreeForm(o map[string]interface{}) FreeForm {
	freeForm := FreeForm{}
	if o != nil {
		j, _ := json.Marshal(o)
		freeForm.json = &j
	}
	return freeForm
}

// UnmarshalJSON implements an alternative parser for this field
func (o *FreeForm) UnmarshalJSON(b []byte) error {
	o.json = &b
	return nil
}

// MarshalJSON specifies how to convert this object into JSON
func (o FreeForm) MarshalJSON() ([]byte, error) {
	if nil == o.json {
		return []byte("{}"), nil
	}
	if len(*o.json) == 0 {
		return []byte("{}"), nil
	}
	return *o.json, nil
}

// IsEmpty determines if the freeform options are empty
func (o FreeForm) IsEmpty() bool {
	if nil == o.json {
		return true
	}
	return len(*o.json) == 0 || string(*o.json) == "{}"
}

// GetMap returns a map created from json
func (o FreeForm) GetMap() (map[string]interface{}, error) {
	m := map[string]interface{}{}
	if nil == o.json {
		return m, nil
	}

	if err := json.Unmarshal(*o.json, &m); err != nil {
		return nil, err
	}
	return m, nil
}
