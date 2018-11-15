package v1alpha1

import (
	"encoding/json"
)

// FreeForm defines a common options parameter to the different structs
type FreeForm struct {
	json []byte
}

// NewFreeForm build a new FreeForm object based on the given map
func NewFreeForm(o map[string]interface{}) FreeForm {
	freeForm := FreeForm{}
	if o != nil {
		freeForm.json, _ = json.Marshal(o)
	}
	return freeForm
}

// UnmarshalJSON implements an alternative parser for this field
func (o *FreeForm) UnmarshalJSON(b []byte) error {
	o.json = b
	return nil
}

// MarshalJSON specifies how to convert this object into JSON
func (o FreeForm) MarshalJSON() ([]byte, error) {
	if len(o.json) == 0 {
		return []byte("{}"), nil
	}
	return o.json, nil
}

// IsEmpty determines if the freeform options are empty
func (o FreeForm) IsEmpty() bool {
	return len(o.json) == 0 || string(o.json) == "{}"
}
