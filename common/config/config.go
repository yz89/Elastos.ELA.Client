package config

import (
	"os"
	"log"
	"bytes"
	"io/ioutil"
	"encoding/json"
)

const (
	CONFIG_FILENAME = "./cli-config.json"
)

var Config *Configuration

type Configuration struct {
	Debug        bool   `json:Debug`
	IpAddress    string `json:IpAddress`
	HttpJsonPort int    `json:"HttpJsonPort"`
}

func init() {
	data, e := ioutil.ReadFile(CONFIG_FILENAME)
	if e != nil {
		log.Fatal("File error: %v\n", e)
		os.Exit(1)
	}
	// Remove the UTF-8 Byte Order Mark
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	Config = new(Configuration)
	e = json.Unmarshal(data, Config)
	if e != nil {
		log.Fatal("Unmarshal json file erro %v", e)
		os.Exit(1)
	}
}
