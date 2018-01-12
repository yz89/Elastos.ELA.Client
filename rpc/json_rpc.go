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

var rpcAddress = getRPCAddress()

func getRPCAddress() string {
	return "http://" + config.Config().IpAddress + ":" + strconv.Itoa(config.Config().HttpJsonPort)
}

func GetCurrentHeight() (uint32, error) {
	result, err := CallAndUnmarshal("getcurrentheight", nil)
	if err != nil {
		return 0, err
	}
	return uint32(result.(float64)), nil
}

func GetBlockByHeight(height uint32) (*BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblock", Param("Height", height))
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	unmarshal(&resp, block)

	return block, nil
}

func Call(method string, params map[string]string) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	//log.Trace("RPC call:", string(data))
	resp, err := http.Post(rpcAddress, "application/json", strings.NewReader(string(data)))
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

func CallAndUnmarshal(method string, params map[string]string) (interface{}, error) {
	body, err := Call(method, params)
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
