package data_test

import (
	"testing"

	"github.com/satriahrh/letter-block/data"
	"github.com/stretchr/testify/assert"
)

func TestNewLetterBank(t *testing.T) {
	t.Run("ErrorNoLanguageFound", func(t *testing.T) {
		_, err := data.NewLetterBank("--")
		assert.EqualError(t, err, data.ErrorNoLanguageFound.Error())
	})
	t.Run("Success", func(t *testing.T) {
		letterBank, err := data.NewLetterBank("id")
		if assert.NoError(t, err) {
			assert.EqualValues(t, []uint8{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1,
				2, 2, 2,
				3, 3, 3, 3,
				4, 4, 4, 4, 4, 4, 4, 4,
				5, 5, 5, 5, 5,
				6, 6, 6,
				7, 7,
				8, 8, 8, 8, 8, 8, 8, 8,
				9,
				10, 10, 10,
				11, 11, 11,
				12, 12, 12,
				13, 13, 13, 13, 13, 13, 13, 13, 13,
				14, 14, 14,
				15, 15,
				17, 17, 17, 17,
				18, 18, 18,
				19, 19, 19, 19, 19,
				20, 20, 20, 20, 20,
				21,
				22,
				24, 24,
				25,
			}, letterBank)
		}
	})
}

func TestLetterBank_Pop(t *testing.T) {
	t.Run("ErrorLetterBankOutOfRange", func(t *testing.T) {
		letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
		_, err := letterBank.Pop(5)
		assert.EqualError(t, err, data.LetterBankOutOfRange.Error())
	})
	t.Run("Success", func(t *testing.T) {
		t.Run("PopEverything", func(t *testing.T) {
			letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
			popOut, err := letterBank.Pop(len(letterBank))
			if assert.NoError(t, err) {
				assert.Equal(t, popOut, []uint8{0, 1, 2, 3})
				assert.EqualValues(t, letterBank, []uint8{})
			}
		})
		t.Run("PopJustEnough", func(t *testing.T) {
			letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
			popOut, err := letterBank.Pop(2)
			if assert.NoError(t, err) {
				assert.Equal(t, popOut, []uint8{0, 1})
				assert.EqualValues(t, letterBank, []uint8{2, 3})
			}
		})
		t.Run("PopNothing", func(t *testing.T) {
			letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
			popOut, err := letterBank.Pop(0)
			if assert.NoError(t, err) {
				assert.Equal(t, popOut, []uint8{})
				assert.EqualValues(t, letterBank, []uint8{0, 1, 2, 3})
			}
		})
	})
}

func TestLetterBank_Shuffle(t *testing.T) {
	original := []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 14, 15}
	toBeLetterBank := make([]uint8, len(original))
	copy(toBeLetterBank, original)
	letterBank := data.LetterBank(toBeLetterBank)
	assert.Equal(t, original, []uint8(letterBank))
	letterBank.Shuffle()
	assert.NotEqual(t, original, []uint8(letterBank))
}
