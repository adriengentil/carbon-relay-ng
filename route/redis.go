package route

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis"
	dest "github.com/graphite-ng/carbon-relay-ng/destination"
	"github.com/graphite-ng/carbon-relay-ng/matcher"
	log "github.com/sirupsen/logrus"
)

type Redis struct {
	baseRoute
	client *redis.Client
}

func NewRedis(key string, prefix string, sub string, regex string) (Route, error) {
	log.Infof("NewRedis key '%s'", key)

	m, err := matcher.New(prefix, sub, regex)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	r := &Redis{
		baseRoute: baseRoute{sync.Mutex{}, atomic.Value{}, key},
		client:    client,
	}

	r.config.Store(baseConfig{*m, make([]*dest.Destination, 0)})

	return r, nil
}

func (r *Redis) Dispatch(buf []byte) {
	log.Infof("Redis.Dispatch: %s", buf)

	elements := strings.Fields(string(buf))
	if len(elements) != 3 {
		return
	}

	r.client.Set(elements[0], strings.Join(elements[1:2], ":"), 0).Err()
}

func (r *Redis) Snapshot() Snapshot {
	return makeSnapshot(&r.baseRoute, "Redis")
}
