package dictionary

type Dictionary interface {
	LemmaIsValid(string) (bool, error)
}
