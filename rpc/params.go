package rpc

import "strconv"

type Params map[string]string

func Param(key string, value interface{}) Params {
	return Params{}.Add(key, value)
}

func (p Params) Add(key string, value interface{}) Params {
	var param string
	switch v := value.(type) {
	case string:
		param = v
	case int:
		param = strconv.FormatInt(int64(v), 10)
	case int16:
		param = strconv.FormatInt(int64(v), 10)
	case int32:
		param = strconv.FormatInt(int64(v), 10)
	case int64:
		param = strconv.FormatInt(int64(v), 10)
	case uint8:
		param = strconv.FormatUint(uint64(v), 10)
	case uint16:
		param = strconv.FormatUint(uint64(v), 10)
	case uint32:
		param = strconv.FormatUint(uint64(v), 10)
	case uint64:
		param = strconv.FormatUint(uint64(v), 10)
	}
	p[key] = param
	return p
}
