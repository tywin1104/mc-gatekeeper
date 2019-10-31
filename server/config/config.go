package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Config struct
type Config struct {
	MongodbConnStr  string   `yaml:"mongodbConn"`
	RabbitmqConnStr string   `yaml:"rabbitMQConn"`
	TaskQueueName   string   `yaml:"taskQueueName"`
	APIPort         string   `yaml:"port"`
	SMTPServer      string   `yaml:"SMTPServer"`
	SMTPPort        int64    `yaml:"SMTPPort"`
	SMTPEmail       string   `yaml:"SMTPEmail"`
	SMTPPassword    string   `yaml:"SMTPPassword"`
	Ops             []string `yaml:",flow"`
	PassPhrase      string   `yaml:"passphrase"`
	JWTTokenSecret  string   `yaml:"jwtTokenSecret"`
	AdminUsername   string   `yaml:"adminUsername"`
	AdminPassword   string   `yaml:"adminPassword"`
}

// LoadConfig loads config file
func LoadConfig() (*Config, error) {
	var appConfig *Config
	file, err := os.Open("config.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &appConfig)
	if err != nil {
		return nil, err
	}
	return appConfig, nil
}
