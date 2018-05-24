package net

import (
	"gopkg.in/dedis/kyber.v2"
)

// ALL_ALL_PARAMETERS message contains all the parameters used by the protocol.
type ALL_ALL_PARAMETERS struct {
	TrusteesPks []kyber.Point // only filled when the relay sends this to the clients
	ForceParams bool
	ParamsInt   map[string]int
	ParamsStr   map[string]string
	ParamsBool  map[string]bool
}

/**
 * Adds a (key, val) to the ALL_ALL_PARAMS message
 */
func (m *ALL_ALL_PARAMETERS) Add(key string, val interface{}) {
	switch typedVal := val.(type) {
	case int:
		if m.ParamsInt == nil {
			m.ParamsInt = make(map[string]int)
		}
		m.ParamsInt[key] = typedVal
	case string:
		if m.ParamsStr == nil {
			m.ParamsStr = make(map[string]string)
		}
		m.ParamsStr[key] = typedVal
	case bool:
		if m.ParamsBool == nil {
			m.ParamsBool = make(map[string]bool)
		}
		m.ParamsBool[key] = typedVal
	}

}

/**
 * From the message, returns the "data[key]" if it exists, or "elseVal"
 */
func (m *ALL_ALL_PARAMETERS) BoolValueOrElse(key string, elseVal bool) bool {
	if val, ok := m.ParamsBool[key]; ok {
		return val
	}
	return elseVal
}

/**
 * From the message, returns the "data[key]" if it exists, or "elseVal"
 */
func (m *ALL_ALL_PARAMETERS) IntValueOrElse(key string, elseVal int) int {
	if val, ok := m.ParamsInt[key]; ok {
		return val
	}
	return elseVal
}

/**
 * From the message, returns the "data[key]" if it exists, or "elseVal"
 */
func (m *ALL_ALL_PARAMETERS) StringValueOrElse(key string, elseVal string) string {
	if val, ok := m.ParamsStr[key]; ok {
		return val
	}
	return elseVal
}
