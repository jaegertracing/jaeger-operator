package upgrade

import (
	"strconv"
)

func flagBoolValue(v interface{}) bool {
	strValue, isString := v.(string)
	if isString {
		value, err := strconv.ParseBool(strValue)
		if err != nil {
			// a default value if I can't parse it
			return false
		}
		return value
	}

	boolValue := v.(bool)
	return boolValue
}
