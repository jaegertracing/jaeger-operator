package upgrade

import "strings"

func flagBoolValue(v interface{}) bool {
	strValue, isString := v.(string)
	if isString {
		if strings.EqualFold(strValue, "true") {
			return true
		}
		return false
	}

	boolValue := v.(bool)
	return boolValue
}
