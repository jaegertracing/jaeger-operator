package util

import (
	"fmt"
)

// Truncate will shorten the length of the instance name so that it contains at most max chars when combined with the fixed part
// If the fixed part is already bigger than the max, this function is noop.
func Truncate(format string, max int, values ...interface{}) string {
	var truncated []interface{}
	result := fmt.Sprintf(format, values...)

	if excess := len(result) - max; excess > 0 {
		// we try to reduce the first string we find
		for _, value := range values {
			if excess == 0 {
				continue
			}

			if s, ok := value.(string); ok {
				if len(s) > excess {
					value = s[:len(s)-excess]
					excess = 0
				} else {
					value = "" // skip this value entirely
					excess = excess - len(s)
				}
			}

			truncated = append(truncated, value)
		}

		result = fmt.Sprintf(format, truncated...)
	}

	// if at this point, the result is still bigger than max, apply a hard cap:
	if len(result) > max {
		return result[:max]
	}

	return result
}
