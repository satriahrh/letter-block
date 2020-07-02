package data

func Letters(language string) (string, error) {
	tile := tiles[language]
	if len(tile.Letters) == 0 {
		return "", ErrorNoLanguageFound
	}

	return tile.Letters, nil
}
