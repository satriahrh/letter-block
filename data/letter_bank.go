package data

import (
	"math/rand"
	"time"
)

type LetterBank []uint8

func NewLetterBank(language string) (LetterBank, error) {
	tiles := tiles[language]
	if len(tiles.Letters) == 0 {
		return nil, ErrorNoLanguageFound
	}
	letterBankCandidate := make([]uint8, 0)
	for i, num := range tiles.Distribution {
		letter := uint8(i + 1)
		for j := 0; j < num; j++ {
			letterBankCandidate = append(letterBankCandidate, letter)
		}
	}

	return LetterBank(letterBankCandidate), nil
}

func (letterBank *LetterBank) Pop(n uint) []uint8 {
	var popOut []uint8
	if uint(len(*letterBank)) < n {
		popOut = *letterBank
		*letterBank = []uint8{}
	} else {
		popOut = (*letterBank)[:n]
		*letterBank = (*letterBank)[n:]
	}
	return popOut
}

func (letterBank *LetterBank) Shuffle() {
	rand.Seed(time.Now().Unix())
	rand.Shuffle(len(*letterBank), func(i, j int) {
		(*letterBank)[i], (*letterBank)[j] = (*letterBank)[j], (*letterBank)[i]
	})
}
