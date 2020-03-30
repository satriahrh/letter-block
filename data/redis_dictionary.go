package data

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type RedisDictionary struct {
	ttl    time.Duration
	client redis.Cmdable
}

func NewRedisDictionary(ttl time.Duration, client redis.Cmdable) RedisDictionary {
	return RedisDictionary{
		client: client,
		ttl:    ttl,
	}
}

func generateKey(lang, key string) string {
	return fmt.Sprintf("%v.%v", lang, key)
}

func (r *RedisDictionary) Get(lang, key string) (bool, bool) {
	dictionaryKey := generateKey(lang, key)
	val, err := r.client.GetBit(dictionaryKey, 1).Result()

	if err != nil {
		return false, false
	}

	return val == 1, true
}

func (r *RedisDictionary) Set(lang, key string, value bool) {
	val := int(0)
	if value {
		val = 1
	}
	r.client.SetBit(generateKey(lang, key), 1, val)
}
