package net

/**
 * From map "data", returns the "data[key]" if it exists, or "elseVal"
 */
func ValueOrElse(data map[string]interface{}, key string, elseVal interface{}) interface{}{
	if val, ok := data[key]; ok {
		return val
	}
	return elseVal
}