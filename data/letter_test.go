package data_test

import (
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/stretchr/testify/assert"
)

func TestLetters(t *testing.T) {
	t.Run("ErrorNoLanguageFound", func(t *testing.T) {
		_, err := data.Letters("--")
		assert.EqualError(t, err, data.ErrorNoLanguageFound.Error())
	})
	t.Run("Success", func(t *testing.T) {
		letters, err := data.Letters("id")
		if assert.NoError(t, err) {
			assert.Equal(t, " abcdefghijklmnopqrstuvwxyz", letters)
		}
	})
}
