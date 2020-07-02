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
			assert.Equal(t, []uint8{
				// a
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				// b
				2, 2, 2, 2,
				// c
				3, 3, 3,
				// d
				4, 4, 4, 4,
				// e
				5, 5, 5, 5, 5, 5, 5, 5,
				// f
				6, 6, 6, 6, 6,
				// g
				7, 7, 7,
				// h
				8, 8,
				// i
				9, 9, 9, 9, 9, 9, 9, 9,
				// j
				10,
				// k
				11, 11, 11,
				// l
				12, 12, 12,
				// m
				13, 13, 13,
				// n
				14, 14, 14, 14, 14, 14, 14, 14, 14,
				// o
				15, 15, 15,
				// p
				16, 16,
				// r
				18, 18, 18, 18,
				// s
				19, 19, 19,
				// t
				20, 20, 20, 20, 20,
				// u
				21, 21, 21, 21, 21,
				// v
				22,
				// w
				23,
				// y
				25, 25,
				// z
				26,
			}, []uint8(letterBank))
		}
	})
}

func TestLetterBank_Pop(t *testing.T) {
	t.Run("PopEverything", func(t *testing.T) {
		letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
		popOut := letterBank.Pop(uint(len(letterBank)))
		assert.Equal(t, popOut, []uint8{0, 1, 2, 3})
		assert.EqualValues(t, letterBank, []uint8{})
	})
	t.Run("PopMore", func(t *testing.T) {
		letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
		popOut := letterBank.Pop(5)
		assert.Equal(t, popOut, []uint8{0, 1, 2, 3})
		assert.EqualValues(t, letterBank, []uint8{})
	})
	t.Run("PopJustEnough", func(t *testing.T) {
		letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
		popOut := letterBank.Pop(2)
		assert.Equal(t, popOut, []uint8{0, 1})
		assert.EqualValues(t, letterBank, []uint8{2, 3})
	})
	t.Run("PopNothing", func(t *testing.T) {
		letterBank := data.LetterBank([]uint8{0, 1, 2, 3})
		popOut := letterBank.Pop(0)
		assert.Equal(t, popOut, []uint8{})
		assert.EqualValues(t, letterBank, []uint8{0, 1, 2, 3})
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
