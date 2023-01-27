package redis

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"github.com/hysios/mx/config"
)

// RedisProvider is a config provider that uses redis as the backend.
type RedisProvider struct {
	rdb  *redis.Client
	Key  string
	vals config.Map
}

type RedisOption struct {
	Addr     string
	Password string
	DB       int
	Key      string
	Mock     *redis.Client
}

// NewRedisProvider returns a new RedisProvider.
func NewRedisProvider(options *RedisOption) (*RedisProvider, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password, // no password set
		DB:       options.DB,
	})

	if options.Mock != nil {
		rdb = options.Mock
	}

	_, err := rdb.Ping(rdb.Context()).Result()
	if err != nil {
		return nil, err
	}

	return &RedisProvider{rdb: rdb, Key: options.Key}, nil
}

// MustRedisProvider returns a new RedisProvider or panic.
func MustRedisProvider(options *RedisOption) *RedisProvider {
	f, err := NewRedisProvider(options)
	if err != nil {
		panic(err)
	}
	return f
}

// load get value from redis
func (p *RedisProvider) load() (val config.Map, ok bool) {
	var ctx = context.Background()
	rslt, err := p.rdb.Get(ctx, p.Key).Result()
	if err != nil {
		return
	}

	val = make(map[string]interface{})

	if err = json.Unmarshal([]byte(rslt), &val); err != nil {
		return nil, false
	}

	return val, true
}

// store set value to redis
func (p *RedisProvider) store(val config.Map) (interface{}, error) {
	var ctx = context.Background()

	b, err := val.JSON()
	if err != nil {
		return nil, err
	}

	rslt, err := p.rdb.Set(ctx, p.Key, b, 0).Result()
	if err != nil {
		return nil, err
	}
	return rslt, nil
}

// LookupPath returns the value of the given selector.
func (p *RedisProvider) LookupPath(selector string) (val *config.Value, ok bool) {
	if p.vals == nil {
		p.vals, ok = p.load()
		if !ok {
			return
		}
	}

	val = p.vals.Get(selector)
	ok = !val.IsNil()
	return
}

// Set sets the value of the given selector.
func (p *RedisProvider) Set(selector string, val interface{}) interface{} {
	if p.vals == nil {
		p.vals, _ = p.load()
		if p.vals == nil {
			p.vals = make(map[string]interface{})
		}
	}
	defer func() {
		if _, err := p.store(p.vals); err != nil {
			panic(err)
		}
	}()

	old := p.vals.Get(selector)
	p.vals.Set(selector, val)
	return old.Data()
}

// Update updates the values of the given map.
func (p *RedisProvider) Update(vals map[string]interface{}) config.Map {
	if p.vals == nil {
		p.vals, _ = p.load()
		if p.vals == nil {
			p.vals = make(map[string]interface{})
		}
	}
	defer p.store(p.vals)

	return p.vals.MergeHere(vals)
}

// Data returns the data of the provider.
func (p *RedisProvider) Data() config.Map {
	if p.vals == nil {
		p.vals, _ = p.load()
		if p.vals == nil {
			p.vals = config.Map{}
		}
	}
	return p.vals
}
