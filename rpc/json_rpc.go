package rpc

import (
	"fmt"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"ELAClient/common/config"
)

var url string

func GetBlockCount() (uint32, error) {
	result, err := CallAndUnmarshal("getblockcount")
	if err != nil {
		return 0, err
	}
	return uint32(result.(float64)), nil
}

func GetBlockByHeight(height uint32) (*BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblock", height)
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	unmarshal(&resp, block)

	return block, nil
}

func Call(method string, params ...interface{}) ([]byte, error) {
	if url == "" {
		url = "http://" + config.Config().IpAddress + ":" + strconv.Itoa(config.Config().HttpJsonPort)
	}
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     88888,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	//log.Trace("RPC call:", string(data))
	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Printf("POST requset: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//log.Trace("RPC resp:", string(body))

	return body, nil
}

func CallAndUnmarshal(method string, params ...interface{}) (interface{}, error) {
	body, err := Call(method, format(params...)...)
	if err != nil {
		return nil, err
	}

	resp := map[string]interface{}{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), nil
	}

	if resp["result"] == nil {
		return "", nil
	}
	return resp["result"], nil
}

func format(params ...interface{}) []interface{} {
	if params == nil {
		return []interface{}{}
	}
	return params
}

func unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}
