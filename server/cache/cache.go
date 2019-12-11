package cache

import (
	"encoding/json"
	"errors"
	"fmt"
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
	statsKey      = "Stats"

	statsPending     = "pending"
	statsDenied      = "denied"
	statsApproved    = "approved"
	statsBanned      = "banned"
	statsDeactivated = "deactivated"
	statsAvgResp     = "averageResponseTimeInMinutes"
	statsTotResp     = "totalResponseTimeInMinutes"
	statsMale        = "maleCount"
	statsFemale      = "femaleCount"
	statsOtherGender = "otherGenderCount"
	statsAgeGroup1   = "ageGroup1Count"
	statsAgeGroup2   = "ageGroup2Count"
	statsAgeGroup3   = "ageGroup3Count"
	statsAgeGroup4   = "ageGroup4Count"
	statsAggregate   = "AggregateStats"

	maxRetry     = 5
	layoutISO    = "01/02 2016"
	ageGroupStep = 15
)

// Service represents a redis cache that is used to cache API results
// and store application specific stats
type Service struct {
	dbService *db.Service
	pool      *redis.Pool
	sseServer *sse.Broker
}

// Stats is composed of both thre real-time stats that got updated in real-time after each
// application status change AND aggregate stats that are analyzed and updated at a regular interval
type Stats struct {
	Pending                      int64          `redis:"pending" json:"pending"`
	Denied                       int64          `redis:"denied" json:"denied"`
	Approved                     int64          `redis:"approved" json:"approved"`
	Banned                       int64          `redis:"banned" json:"banned"`
	Deactivated                  int64          `redis:"deactivated" json:"deactivated"`
	AverageResponseTimeInMinutes float64        `redis:"averageResponseTimeInMinutes" json:"averageResponseTimeInMinutes"`
	TotalResponseTimeInMinutes   float64        `redis:"totalResponseTimeInMinutes" json:"totalResponseTimeInMinutes"`
	MaleCount                    int64          `redis:"maleCount" json:"maleCount"`
	FemaleCount                  int64          `redis:"femaleCount" json:"femaleCount"`
	OtherGenderCount             int64          `redis:"otherGenderCount" json:"otherGenderCount"`
	AgeGroup1Count               int64          `redis:"ageGroup1Count" json:"ageGroup1Count"`
	AgeGroup2Count               int64          `redis:"ageGroup2Count" json:"ageGroup2Count"`
	AgeGroup3Count               int64          `redis:"ageGroup3Count" json:"ageGroup3Count"`
	AgeGroup4Count               int64          `redis:"ageGroup4Count" json:"ageGroup4Count"`
	AggregateStats               AggregateStats `redis:"-" json:"aggregateStats"`
}

// AggregateStats are records of some time-consuming results and got updated at regular intervals
type AggregateStats struct {
	OvertimeCount      int                     `json:"overtimeCount"`
	AdminPerformance   map[string]*Performance `json:"adminPerformance"`
	DivergentCount     int                     `json:"divergentCount"`
	DivergentUsernames []string                `json:"divergentUsernames"`
}

// Performance contains stats information about each ops
type Performance struct {
	TotalHandled                 int     `json:"totalHandled"`
	AverageResponseTimeInMinutes float64 `json:"averageResponseTimeInMinutes"`
	totalResponseTimeInMinutes   float64
}

var log = logrus.New()

// NewService creates and initilize a new caching service
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

// UpdateAggregateStats will be called at certain time intervals to start calculate and analyze all records
// and update the aggregateStats field in the Stats cache
func (svc *Service) UpdateAggregateStats() error {
	overtimeCount := 0

	pendingRequests, err := svc.dbService.GetRequests(-1, bson.M{"status": types.StatusPending})
	if err != nil {
		return err
	}
	currentTime := time.Now()
	for _, pendingRequest := range pendingRequests {
		// check for overtime
		if currentTime.Sub(pendingRequest.Timestamp).Hours() >= 24 {
			overtimeCount++
		}
	}
	fulfilledRequests, err := svc.dbService.GetRequests(-1, bson.M{
		"status": bson.M{"$in": []string{types.StatusDenied, types.StatusApproved, types.StatusBanned, types.StatusDeactivated}},
	})
	if err != nil {
		return err
	}
	adminPerformance := make(map[string]*Performance)
	divergentCount := 0
	divergentUsernames := []string{}
	matchedStatus := map[string]string{
		types.StatusApproved:    types.OnserverWhitelisted,
		types.StatusDenied:      types.OnserverNone,
		types.StatusDeactivated: types.OnserverNone,
		types.StatusBanned:      types.OnserverBanned,
	}
	for _, request := range fulfilledRequests {
		// analyze admin performance
		processingTime := request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
		if p, ok := adminPerformance[request.Admin]; ok {
			p.totalResponseTimeInMinutes += processingTime
			p.AverageResponseTimeInMinutes = p.totalResponseTimeInMinutes / (float64(p.TotalHandled) + 1)
			p.TotalHandled++
		} else {
			p := new(Performance)
			p.TotalHandled = 1
			p.totalResponseTimeInMinutes = processingTime
			p.AverageResponseTimeInMinutes = processingTime
			adminPerformance[request.Admin] = p
		}
		// If the user's on-server status does not match its desired status for over a certain time, consider it divergent
		if request.OnserverStatus != matchedStatus[request.Status] && time.Now().Sub(request.LastUpdatedTimestamp).Minutes() >= 2 {
			divergentCount++
			divergentUsernames = append(divergentUsernames, request.Username)
		}
	}
	var aggreagateStats = AggregateStats{
		OvertimeCount:      overtimeCount,
		AdminPerformance:   adminPerformance,
		DivergentCount:     divergentCount,
		DivergentUsernames: divergentUsernames,
	}
	// serialize objects to JSON
	json, err := json.Marshal(aggreagateStats)
	if err != nil {
		return err
	}
	conn := svc.pool.Get()
	defer conn.Close()
	_, err = conn.Do("HMSET", statsKey, statsAggregate, json)
	if err != nil {
		return err
	}
	err = svc.BroadcastStats()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("Unable to broadcast event for aggregate stats update")
	}
	return nil
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

// UpdateAllRequests updates the cached value of all requests by fetching from db once
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

// getStats get both real-time and aggregate stats from cache and unmarshal into struct
func (svc *Service) getStats() (Stats, error) {
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
	// Need to manually unmarshal AggregateStats as it is a nested struct
	value, err := redis.Values(conn.Do("HMGET", statsKey, statsAggregate))
	// redis.Values returns []interface{}
	aggregateStatsStr := fmt.Sprintf("%s", value[0])
	if err != nil {
		return Stats{}, err
	}
	var aggregateStats AggregateStats
	err = json.Unmarshal([]byte(aggregateStatsStr), &aggregateStats)
	if err != nil {
		return Stats{}, err
	}
	stats.AggregateStats = aggregateStats
	return stats, nil
}

// UpdateRealTimeStats makes proper change to the real-time portion of the stats in the cache
// depending on changes on the system
func (svc *Service) UpdateRealTimeStats(request types.WhitelistRequest) error {
	for n := 1; n <= maxRetry; n++ {
		conn := svc.pool.Get()
		defer conn.Close()
		stats, err := svc.getStats()
		if err != nil {
			return err
		}
		// Instruct Redis to watch the stats hash for any changes
		_, err = conn.Do("WATCH", statsKey)
		if err != nil {
			return err
		}
		var args = make([]interface{}, 0)
		args = append(args, statsKey)
		newApprovedCount := stats.Approved
		newDeniedCount := stats.Denied
		newPendingCount := stats.Pending
		newBannedCount := stats.Banned
		newDeactivatedCount := stats.Deactivated
		newTotalResponseTimeInMinutes := stats.TotalResponseTimeInMinutes
		var newAverageResponseTimeInMinutes float64

		// Use the MULTI command to inform Redis that we are starting
		// a new transaction.
		err = conn.Send("MULTI")
		if err != nil {
			return err
		}
		// Update the values for stats on the cache according to different type of actions being
		// made for the request
		switch request.Status {
		case types.StatusApproved:
			updateAgeGenderStats(request, conn, 1)
			newApprovedCount++
			newPendingCount--
			newTotalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			args = append(args, []interface{}{statsPending, newPendingCount, statsApproved, newApprovedCount, statsTotResp, newTotalResponseTimeInMinutes}...)
		case types.StatusDenied:
			newDeniedCount++
			newPendingCount--
			newTotalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			args = append(args, []interface{}{statsPending, newPendingCount, statsDenied, newDeniedCount, statsTotResp, newTotalResponseTimeInMinutes}...)
		case types.StatusPending:
			newPendingCount++
			args = append(args, []interface{}{statsPending, newPendingCount}...)
		case types.StatusBanned:
			updateAgeGenderStats(request, conn, 1)
			newBannedCount++
			newApprovedCount--
			args = append(args, []interface{}{statsApproved, newApprovedCount, statsBanned, newBannedCount}...)
		case types.StatusDeactivated:
			updateAgeGenderStats(request, conn, 1)
			newApprovedCount--
			newDeactivatedCount++
			args = append(args, []interface{}{statsApproved, newApprovedCount, statsDeactivated, newDeactivatedCount}...)
		}
		// Only update the average reponse time stats if the request is being fulfilled
		if newTotalResponseTimeInMinutes != 0 {
			newAverageResponseTimeInMinutes = newTotalResponseTimeInMinutes / float64(newApprovedCount+newDeniedCount+newBannedCount+newDeactivatedCount)
			args = append(args, []interface{}{statsAvgResp, newAverageResponseTimeInMinutes}...)
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
			log.Debugf("Race condition detected during stats update. Retring %d/%d \n", n, maxRetry)
			time.Sleep(time.Second * 2)
			continue
		} else if err != nil {
			return err
		}
		// After a successful update, broadcast the new stats to clients
		// who are listening for the stats update via ServerSideEvent http server
		err = svc.BroadcastStats()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Unable to broadcast event for stats update")
		}
		return nil
	}
	return errors.New("Unable to update stats. Give up")
}

// BroadcastStats will push the current state of stats in cache to clients listening for SSE
func (svc *Service) BroadcastStats() error {
	stats, err := svc.getStats()
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

// SyncStats will run once during startup to synchronize/ initilize everything stats related
func (svc *Service) SyncStats() error {
	// Sync all requets from db to cache
	err := svc.UpdateAllRequests()
	if err != nil {
		return err
	}
	// Sync aggregate stats
	err = svc.UpdateAggregateStats()
	if err != nil {
		return err
	}
	// Sync real-time stats
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
		var approved, denied, pending, banned, deactivated int64
		var totalResponseTimeInMinutes float64
		var maleCount, femaleCount, otherGenderCount int64
		var ageGroup1Count, ageGroup2Count, ageGroup3Count, ageGroup4Count int64
		for _, request := range requests {
			switch request.Status {
			case types.StatusApproved:
				approved++
				// Gather gender metric
				switch request.Gender {
				case "male":
					maleCount++
				case "female":
					femaleCount++
				default:
					otherGenderCount++
				}
				// Gather age group metric
				age := request.Age
				// temp value
				var step int64 = ageGroupStep
				if 0 <= age && age < step {
					ageGroup1Count++
				} else if step <= age && age < step*2 {
					ageGroup2Count++
				} else if step*2 <= age && age < step*3 {
					ageGroup3Count++
				} else {
					ageGroup4Count++
				}
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			case types.StatusDenied:
				denied++
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			case types.StatusPending:
				pending++
			case types.StatusBanned:
				banned++
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			case types.StatusDeactivated:
				deactivated++
				totalResponseTimeInMinutes += request.ProcessedTimestamp.Sub(request.Timestamp).Minutes()
			}
		}
		var averageResponseTimeInMinutes float64
		// Only update the averageResponseTime if there are fulfilled requests
		if totalResponseTimeInMinutes != 0 {
			averageResponseTimeInMinutes = totalResponseTimeInMinutes / float64(approved+denied+banned+deactivated)
		}

		err = conn.Send("MULTI")
		if err != nil {
			return err
		}
		err = conn.Send(
			"HMSET", statsKey,
			statsPending, pending, statsDenied, denied,
			statsApproved, approved,
			statsBanned, banned,
			statsDeactivated, deactivated,
			statsAvgResp, averageResponseTimeInMinutes,
			statsTotResp, totalResponseTimeInMinutes,
			statsMale, maleCount,
			statsFemale, femaleCount,
			statsOtherGender, otherGenderCount,
			statsAgeGroup1, ageGroup1Count,
			statsAgeGroup2, ageGroup2Count,
			statsAgeGroup3, ageGroup3Count,
			statsAgeGroup4, ageGroup4Count)
		if err != nil {
			return err
		}
		_, err = redis.Values(conn.Do("EXEC"))
		if err == redis.ErrNil {
			log.Debugf("Race condition detected during initial cache sync. Retring %d/%d \n", n, maxRetry)
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

// updateAgeGenderStats takes in a reuqest and make appropriate change to the stats
func updateAgeGenderStats(request types.WhitelistRequest, conn redis.Conn, delta int64) {
	switch request.Gender {
	case "male":
		conn.Send("HINCRBY", statsKey, statsMale, delta)
	case "female":
		conn.Send("HINCRBY", statsKey, statsFemale, delta)
	default:
		conn.Send("HINCRBY", statsKey, statsOtherGender, delta)
	}

	// Update age group metric
	age := request.Age
	var step int64 = ageGroupStep
	if 0 <= age && age < step {
		conn.Send("HINCRBY", statsKey, statsAgeGroup1, delta)
	} else if step <= age && age < step*2 {
		conn.Send("HINCRBY", statsKey, statsAgeGroup2, delta)
	} else if step*2 <= age && age < step*3 {
		conn.Send("HINCRBY", statsKey, statsAgeGroup3, delta)
	} else {
		conn.Send("HINCRBY", statsKey, statsAgeGroup4, delta)
	}
}
