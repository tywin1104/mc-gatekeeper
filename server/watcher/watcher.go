package watcher

import (
	"errors"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tywin1104/mc-gatekeeper/broker"
)

// ValidateConfig will validate the config values at initial start and on each subsequent config change events
func ValidateConfig() error {
	// TODO: add more constraints to fail fast
	strategy := viper.GetString("dispatchingStrategy")
	if strategy != "Broadcast" && strategy != "Random" {
		return errors.New("Invalid configuration. Allowed values for dispatchingStrategy: [Broadcast, Random]")
	}
	if strategy == "Random" && viper.GetInt("randomDispatchingThreshold") > len(viper.GetStringSlice("ops")) {
		return errors.New("Invalid configuration. Threshold value for random dispatching can not exceed total number of ops")
	}
	return nil
}

// WatchConfig will watch for config file update and re-validate config values
func WatchConfig(log *logrus.Logger) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Re-validate config each time it changes
		err := ValidateConfig()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Invalid configuration. The application will not not work properly.")
		}
		log.WithFields(logrus.Fields{
			"file": e.Name,
		}).Info("Config file changed:")
	})
}

// WatchForBrokerReconnect will monitor broker service for connection disruption and
// re-establish a new connection and channel
func WatchForBrokerReconnect(broker *broker.Service) {
	broker.WatchForReconnect()
}
