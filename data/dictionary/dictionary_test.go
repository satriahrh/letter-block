package dictionary_test

import (
	"github.com/satriahrh/letter-block/data/dictionary"

	"errors"
	"testing"
	"time"

	"github.com/elliotchance/redismock"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func suiteDictionary() (dict *dictionary.Dictionary, clientMock *redismock.ClientMock) {
	clientMock = redismock.NewMock()
	dict = dictionary.NewDictionary(time.Hour, clientMock)
	return
}

func TestDictionary_DictionaryGet(t *testing.T) {
	lang := "id-id"
	key := "word"
	dictionaryKey := "id-id.word"

	t.Run("ValidAndExist", func(t *testing.T) {
		dict, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(1, nil),
			)

		result, exist := dict.Get(lang, key)

		assert.True(t, result, "invalid")
		assert.True(t, exist, "exist")
	})
	t.Run("InvalidAndExist", func(t *testing.T) {
		dict, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(0, nil),
			)

		result, exist := dict.Get(lang, key)

		assert.False(t, result, "invalid")
		assert.True(t, exist, "exist")
	})
	t.Run("UnexpectedErrorOrNotExisted", func(t *testing.T) {
		dict, clientMock := suiteDictionary()

		clientMock.
			On("GetBit", dictionaryKey, int64(1)).
			Return(
				redis.NewIntResult(0, errors.New("something")),
			)

		result, exist := dict.Get(lang, key)

		assert.False(t, result, "invalid")
		assert.False(t, exist, "not existed")
	})
}

func TestDictionary_DictionarySet(t *testing.T) {
	lang := "id-id"
	key := "word"
	dictionaryKey := "id-id.word"

	t.Run("Valid", func(t *testing.T) {
		dict, clientMock := suiteDictionary()

		clientMock.
			On("SetBit", dictionaryKey, int64(1), 1).
			Return(redis.NewIntResult(0, nil))

		dict.Set(lang, key, true)
	})
	t.Run("Invalid", func(t *testing.T) {
		dict, clientMock := suiteDictionary()

		clientMock.
			On("SetBit", dictionaryKey, int64(1), 0).
			Return(redis.NewIntResult(0, nil))

		dict.Set(lang, key, false)
	})
}
