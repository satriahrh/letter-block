package id_id

import (
	"github.com/satriahrh/letter-block/data"

	"errors"
	"fmt"
	"net/http"
)

var (
	ErrorHttpUnexpected = errors.New("unexpected http error")
)

const (
	baseUrl  = "https://kbbi.kemdikbud.go.id/entri"
	language = "id-id"
)

type IdId struct {
	cache      data.Dictionary
	httpClient *http.Client
}

func NewIdId(dictionary data.Dictionary, httpClient *http.Client) *IdId {
	return &IdId{
		cache:      dictionary,
		httpClient: httpClient,
	}
}

func (d *IdId) LemmaIsValid(lemma string) (result bool, err error) {
	// Exist On Cache?
	var exist bool
	result, exist = d.cache.Get(language, lemma)
	if exist {
		return result, nil
	}

	// Request To KBBI
	url := fmt.Sprintf("%v/%v", baseUrl, lemma)
	res, err := d.httpClient.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = ErrorHttpUnexpected
		return
	}

	return
}
