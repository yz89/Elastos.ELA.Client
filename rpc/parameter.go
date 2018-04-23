package rpc

type Parameter map[string]interface{}

func Param(key string, value interface{}) Parameter {
	return Parameter{}.Add(key, value)
}

func (param Parameter) Add(key string, value interface{}) Parameter {
	param[key] = value
	return param
}
