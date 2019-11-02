package config

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Config struct
type Config struct {
	MongodbConnStr             string   `yaml:"mongodbConn"`
	RabbitmqConnStr            string   `yaml:"rabbitMQConn"`
	TaskQueueName              string   `yaml:"taskQueueName"`
	APIPort                    string   `yaml:"port"`
	SMTPServer                 string   `yaml:"SMTPServer"`
	SMTPPort                   int64    `yaml:"SMTPPort"`
	SMTPEmail                  string   `yaml:"SMTPEmail"`
	SMTPPassword               string   `yaml:"SMTPPassword"`
	Ops                        []string `yaml:",flow"`
	PassPhrase                 string   `yaml:"passphrase"`
	JWTTokenSecret             string   `yaml:"jwtTokenSecret"`
	AdminUsername              string   `yaml:"adminUsername"`
	AdminPassword              string   `yaml:"adminPassword"`
	DispatchingStrategy        string   `yaml:"dispatchingStrategy"`
	RandomDispatchingThreshold int      `yaml:"randomDispatchingThreshold"`
	RecaptchaPrivateKey        string   `yaml:"recaptchaPrivateKey"`
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
	// Pre running sanity configuration check
	if appConfig.DispatchingStrategy != "Broadcast" && appConfig.DispatchingStrategy != "Random" {
		return nil, errors.New("Invalid configuration. Allowed values for dispatchingStrategy: [Broadcast, Random]")
	}
	if appConfig.DispatchingStrategy == "Random" && appConfig.RandomDispatchingThreshold > len(appConfig.Ops) {
		return nil, errors.New("Invalid configuration. Threshold value for random dispatching can not exceed total number of ops")
	}
	return appConfig, nil
}
