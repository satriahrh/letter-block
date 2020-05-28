package data

import (
	"errors"
	"math/rand"
	"time"
)

type LetterBank []uint8

var LetterBankOutOfRange = errors.New("out of range")

func NewLetterBank(language string) (LetterBank, error) {
	tiles := tiles[language]
	if len(tiles.Letters) == 0 {
		return nil, ErrorNoLanguageFound
	}
	letterBankCandidate := make([]uint8, 0)
	for letter, num := range tiles.Distribution {
		for i := 0; i < num; i++ {
			letterBankCandidate = append(letterBankCandidate, uint8(letter))
		}
	}

	return LetterBank(letterBankCandidate), nil
}

func (letterBank *LetterBank) Pop(n int) ([]uint8, error) {
	if len(*letterBank) < n {
		return []uint8{}, errors.New("out of range")
	}
	popOut := (*letterBank)[:n]
	*letterBank = (*letterBank)[n:]
	return popOut, nil
}

func (letterBank *LetterBank) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(*letterBank), func(i, j int) {
		(*letterBank)[i], (*letterBank)[j] = (*letterBank)[j], (*letterBank)[i]
	})
}
