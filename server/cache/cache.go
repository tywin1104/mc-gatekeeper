package cache

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/spf13/viper"
	"github.com/tywin1104/mc-gatekeeper/server/sse"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-gatekeeper/db"
	"github.com/tywin1104/mc-gatekeeper/types"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	allRequestKey = "AllRequests"
	statsKey      = "RequestsStats"
	maxRetry      = 5
	layoutISO     = "01/02 2016"
)

// Service represents a redis cache that is used to cache API results
// and store real-time stats for all applications
type Service struct {
	dbService *db.Service
	pool      *redis.Pool
	sseServer *sse.Broker
}

// Stats represents real-time stats for all application any any moment
// Those are the application specific values that feed to the mgmt dashboard
type Stats struct {
	Pending                      int64   `redis:"pending" json:"pending"`
	Denied                       int64   `redis:"denied" json:"denied"`
	Approved                     int64   `redis:"approved" json:"approved"`
	AverageResponseTimeInMinutes float64 `redis:"averageResponseTimeInMinutes" json:"averageResponseTimeInMinutes"`
	TotalResponseTimeInMinutes   float64 `redis:"totalResponseTimeInMinutes" json:"totalResponseTimeInMinutes"`
}

var log = logrus.New()

// NewService create and initilize a new caching service
func NewService(db *db.Service, sseServer *sse.Broker) *Service {
	var log = logrus.New()
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", viper.GetString("redisConn"))
			return c, err
		},
	}
	// Ping the cache first to verify connection
	conn := pool.Get()
	_, err := conn.Do("PING")
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Unable to connect to redis cache")
	}
	log.Info("Redis cache connection established. Please wait a few moments for initilization...")

	return &Service{
		dbService: db,
		pool:      pool,
		sseServer: sseServer,
	}
}

// GetAllRequests get the cached value of all requets in db if exists
func (svc *Service) GetAllRequests() ([]types.WhitelistRequest, error) {
	conn := svc.pool.Get()
	defer conn.Close()
	// Check if the key exists
	exists, err := redis.Int(conn.Do("EXISTS", allRequestKey))
	if err != nil {
		return nil, err
	} else if exists == 0 {
		return nil, errors.New("Key does not exist")
	}

	// If exists, get cached value
	s, err := redis.String(conn.Do("GET", allRequestKey))
	if err != nil {
		return nil, err
	}
	var requests []types.WhitelistRequest
	if err := json.Unmarshal([]byte(s), &requests); err != nil {
		return nil, err
	}
	return requests, nil
}

// UpdateAllRequests updates the cached value by fetching from db once
func (svc *Service) UpdateAllRequests() error {
	requests, err := svc.dbService.GetRequests(-1, bson.D{{}})
	if err != nil {
		return err
	}
	// serialize objects to JSON
	json, err := json.Marshal(requests)
	if err != nil {
		return err
	}
	conn := svc.pool.Get()
	_, err = conn.Do("SET", allRequestKey, json)
	if err != nil {
		return err
	}
	return nil
}

// GetStats get the real-time stats from cache
func (svc *Service) GetStats() (Stats, error) {
	conn := svc.pool.Get()
	defer conn.Close()
	values, err := redis.Values(conn.Do("HGETALL", statsKey))
	if err != nil {
		return Stats{}, err
	}

	var stats Stats
	err = redis.ScanStruct(values, &stats)
	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}

// UpdateStats makes proper change to the stats exposed to the mgmt dashboard according
// to the current status of the request
func (svc *Service) UpdateStats(request types.WhitelistRequest) error {
	for n := 1; n <= maxRetry; n++ {
		conn := svc.pool.Get()
		defer conn.Close()
		stats, err := svc.GetStats()
		if err != nil {
			return err
		}
		// Instruct Redis to watch the stats hash for any changes
		_, err = conn.Do("WATCH", statsKey)
		if err != nil {
			return err
		}
		oldApprovedCount := stats.Approved
		oldDeniedCount := stats.Denied
		oldPendingCount := stats.Pending
		oldTotalResponseTimeInMinutes := stats.TotalResponseTimeInMinutes
		var newApprovedCount, newDeniedCount, newPendingCount int64
		var newTotalResponseTimeInMinutes, newAverageResponseTimeInMinutes float64
		var args = make([]interface{}, 0)
		args = append(args, statsKey)
		// Update the values for stats on the cache according to different type of actions being
		// made for the request
		switch request.Status {
		case "Approved":
			newApprovedCount = oldApprovedCount + 1
			newPendingCount = oldPendingCount - 1
			newTotalResponseTimeInMinutes = oldTotalResponseTimeInMinutes + request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			args = append(args, []interface{}{"pending", newPendingCount, "approved", newApprovedCount, "totalResponseTimeInMinutes", newTotalResponseTimeInMinutes}...)
		case "Denied":
			newDeniedCount = oldDeniedCount + 1
			newPendingCount = oldPendingCount - 1
			newTotalResponseTimeInMinutes = oldTotalResponseTimeInMinutes + request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			args = append(args, []interface{}{"pending", newPendingCount, "denied", newDeniedCount, "totalResponseTimeInMinutes", newTotalResponseTimeInMinutes}...)
		case "Pending":
			newPendingCount = oldPendingCount + 1
			args = append(args, []interface{}{"pending", newPendingCount}...)
		}
		// Only update the average reponse time stats if the request is being fulfilled
		if newTotalResponseTimeInMinutes != 0 {
			newAverageResponseTimeInMinutes = math.Round(newTotalResponseTimeInMinutes / float64(newApprovedCount+newDeniedCount))
			args = append(args, []interface{}{"averageResponseTimeInMinutes", newAverageResponseTimeInMinutes}...)
		}
		// Use the MULTI command to inform Redis that we are starting
		// a new transaction.
		err = conn.Send("MULTI")
		if err != nil {
			return err
		}
		err = conn.Send("HMSET", args...)
		if err != nil {
			return err
		}
		// Execute the transaction. Importantly, use the redis.ErrNil
		// type to check whether the reply from EXEC was nil or not. If
		// it is nil it means that another client changed the WATCHed
		// field, so we use the continue command to re-run
		// the loop.
		_, err = redis.Values(conn.Do("EXEC"))
		if err == redis.ErrNil {
			log.Infof("Race condition detected during stats update. Retring %d/%d \n", n, maxRetry)
			time.Sleep(time.Second * 2)
			continue
		} else if err != nil {
			return err
		}
		// After a successful update, broadcast the new stats to clients
		// who are listening for the stats update via ServerSideEvent http server
		err = svc.BroadcastViaSSE()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Unable to broadcast event for stats update")
		}
		return nil
	}
	return errors.New("Unable to update stats. Give up")
}

// BroadcastViaSSE will push the updated real-time stats data to the client via SeverSideEnvent
func (svc *Service) BroadcastViaSSE() error {
	stats, err := svc.GetStats()
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	svc.sseServer.Notifier <- jsonBytes
	return nil
}

// SyncCache sync cache value with current db status. Should run once during startup
func (svc *Service) SyncCache() error {

	// Sync all requets from db to cache
	err := svc.UpdateAllRequests()
	if err != nil {
		return err
	}
	for n := 1; n <= maxRetry; n++ {
		conn := svc.pool.Get()
		defer conn.Close()
		requests, err := svc.GetAllRequests()
		if err != nil {
			return err
		}
		_, err = conn.Do("WATCH", statsKey)
		if err != nil {
			return err
		}
		// Recalculate the stats for all requests at the moment
		// And update the stats value in cache
		var approved, denied, pending int64
		var totalResponseTimeInMinutes float64
		for _, request := range requests {
			// Gather count related stats
			switch request.Status {
			case "Approved":
				approved++
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			case "Denied":
				denied++
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			case "Pending":
				pending++
			}
		}
		var averageResponseTimeInMinutes float64
		// Only update the averageResponseTime if there are fulfilled requests
		if totalResponseTimeInMinutes != 0 {
			averageResponseTimeInMinutes = math.Round(totalResponseTimeInMinutes / float64(approved+denied))
		}

		err = conn.Send("MULTI")
		if err != nil {
			return err
		}
		err = conn.Send(
			"HMSET", statsKey,
			"pending", pending, "denied", denied,
			"approved", approved,
			"averageResponseTimeInMinutes", averageResponseTimeInMinutes,
			"totalResponseTimeInMinutes", totalResponseTimeInMinutes)
		if err != nil {
			return err
		}
		_, err = redis.Values(conn.Do("EXEC"))
		if err == redis.ErrNil {
			log.Infof("Race condition detected during initial cache sync. Retring %d/%d \n", n, maxRetry)
			time.Sleep(time.Second * 2)
			continue
		} else if err != nil {
			return err
		}
		log.Info("Initial cache sync completed")
		return nil
	}
	return errors.New("Unable to sync cache. Give up")
}
