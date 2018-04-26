package rpc

type Params map[string]interface{}

func Param(key string, value interface{}) Params {
	return Params{}.Add(key, value)
}

func (p Params) Add(key string, value interface{}) Params {
	p[key] = value
	return p
}
