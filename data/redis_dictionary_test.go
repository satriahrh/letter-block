package data_test

import (
	"github.com/satriahrh/letter-block/data"

	"errors"
	"testing"
	"time"

	"github.com/elliotchance/redismock"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func suiteDictionary() (redisDictionary data.RedisDictionary, clientMock *redismock.ClientMock) {
	clientMock = redismock.NewMock()
	redisDictionary = data.NewRedisDictionary(time.Hour, clientMock)
	return
}

func TestRedisDictionary_DictionaryGet(t *testing.T) {
	lang := "id-id"
	key := "word"
	dictionaryKey := "id-id.word"

	t.Run("ValidAndExist", func(t *testing.T) {
		redisDictionary, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(1, nil),
			)

		result, exist := redisDictionary.Get(lang, key)

		assert.True(t, result, "invalid")
		assert.True(t, exist, "exist")
	})
	t.Run("InvalidAndExist", func(t *testing.T) {
		redisDictionary, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(0, nil),
			)

		result, exist := redisDictionary.Get(lang, key)

		assert.False(t, result, "invalid")
		assert.True(t, exist, "exist")
	})
	t.Run("UnexpectedErrorOrNotExisted", func(t *testing.T) {
		redisDictionary, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(0, errors.New("something")),
			)

		result, exist := redisDictionary.Get(lang, key)

		assert.False(t, result, "invalid")
		assert.False(t, exist, "not existed")
	})
}

func TestRedisDictionary_DictionarySet(t *testing.T) {
	lang := "id-id"
	key := "word"
	dictionaryKey := "id-id.word"

	t.Run("Valid", func(t *testing.T) {
		redisDictionary, clientMock := suiteDictionary()

		clientMock.
			On("SetBit", dictionaryKey, int64(1), 1).
			Return(redis.NewIntResult(0, nil))

		redisDictionary.Set(lang, key, true)
	})
	t.Run("Invalid", func(t *testing.T) {
		redisDictionary, clientMock := suiteDictionary()

		clientMock.
			On("SetBit", dictionaryKey, int64(1), 0).
			Return(redis.NewIntResult(0, nil))

		redisDictionary.Set(lang, key, false)
	})
}
