package server

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// HandleGetAggregateStats will return cached aggregate stats that is updated at regular interval
func (svc *Service) HandleGetAggregateStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		var msg map[string]interface{}
		// Try to fetch value from cache first
		stats, err := svc.cache.GetAggregateStats()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Unable to get aggregate stats")
		} else {
			msg = map[string]interface{}{"stats": stats}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(msg)
		}
	}
}
