package net

/**
 * From map "data", returns the "data[key]" if it exists, or "elseVal"
 */
func ValueOrElse(data map[string]Interface, key string, elseVal interface{}) interface{} {
	if val, ok := data[key]; ok {
		return val.Data
	}
	return elseVal
}

/**
 * Converts from map[string]interface{} to map[string]Interface
 */
func MapInterface(data map[string]interface{}) map[string]Interface {
	res := make(map[string]Interface)
	for k, v := range data {
		res[k] = Interface{Data: v}
	}
	return res
}