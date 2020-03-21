package runner

import (
	"gopkg.in/yaml.v2"
	"hseSQL/internal/database"
	"io/ioutil"
	"os"
)

type Config struct {
	ServerAddr string   `yaml:"server_addr"`
	DbConfig *database.Config `yaml:"db_config"`
}

func ReadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}


