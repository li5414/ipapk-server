package conf

import (
	"encoding/json"
	"fmt"
	"github.com/li5414/ipapk-server/utils"
	"io/ioutil"
)

var AppConfig *Config

type Config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	RemoteHost string `json:"remote"`
	RemotePort string `json:"remotePort"`
	Proxy    string `json:"proxy"`
	Database string `json:"database"`
}

func InitConfig(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &AppConfig); err != nil {
		return err
	}
	return nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%v:%v", c.Host, c.Port)
}

func (c *Config) RemoteAddr() string {
	return fmt.Sprintf("https://%v:%v", c.RemoteHost, c.RemotePort)
}

func (c *Config) ProxyURL() string {
	if c.Proxy == "" {
		localIp, err := utils.LocalIP()
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("https://%v:%v", localIp.String(), c.Port)
	}
	return c.Proxy
}

func (c *Config) BaseURL() string {
	localIp, err := utils.LocalIP()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("https://%v:%v", localIp.String(), c.Port)
}