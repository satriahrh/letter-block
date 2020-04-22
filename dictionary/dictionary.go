package dictionary

// Dictionary interface of dictionary
type Dictionary interface {
	LemmaIsValid(string) (bool, error)
}
