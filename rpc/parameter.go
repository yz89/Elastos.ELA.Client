package rpc

import "strconv"

type Parameter map[string]string

func Param(key string, value interface{}) Parameter {
	return Parameter{}.Add(key, value)
}

func (param Parameter) Add(key string, value interface{}) Parameter {
	switch value.(type) {
	case int:
		value = strconv.Itoa(value.(int))
	case uint32:
		value = strconv.FormatUint(uint64(value.(uint32)), 10)
	case uint64:
		value = strconv.FormatUint(value.(uint64), 10)
	}
	param[key] = value.(string)
	return param
}
