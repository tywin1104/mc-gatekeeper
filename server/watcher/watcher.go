package watcher

import (
	"errors"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tywin1104/mc-gatekeeper/cache"
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

// AggregateStats will compute and update the cache value for aggregate stats periodically
func AggregateStats(cache *cache.Service, log *logrus.Logger) {
	for range time.Tick(60 * time.Second) {
		go func() {
			err := cache.UpdateAggregateStats()
			if err != nil {
				log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Error("Unable to aggregate stats")
			} else {
				log.Info("Aggregate stats data completed")
			}
		}()
	}
}
