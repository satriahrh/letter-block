package idid

import (
	"github.com/satriahrh/letter-block/data"

	"errors"
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

var (
	// ErrorHTTPUnexpected for any unexpected http error
	ErrorHTTPUnexpected = errors.New("unexpected http error")
)

const (
	baseURL  = "https://kbbi.kemdikbud.go.id/entri"
	language = "id-id"
)

// Dictionary implementation of dictionary of Indonesia
type Dictionary struct {
	cache      data.Dictionary
	httpClient *http.Client
}

// NewDictionary constructor of IdId
func NewDictionary(dictionary data.Dictionary, httpClient *http.Client) *Dictionary {
	return &Dictionary{
		cache:      dictionary,
		httpClient: httpClient,
	}
}

// LemmaIsValid validate lemma through KBBI daring
func (d *Dictionary) LemmaIsValid(lemma string) (result bool, err error) {
	// Exist On Cache?
	var exist bool
	result, exist = d.cache.Get(language, lemma)
	if exist {
		return result, nil
	}

	// Request To KBBI
	url := fmt.Sprintf("%v/%v", baseURL, lemma)
	res, err := d.httpClient.Get(url)
	if err != nil {
		return
	}
	if res.StatusCode != 200 {
		err = ErrorHTTPUnexpected
		return
	}

	defer func() {
		_ = res.Body.Close()
	}()
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	// Find the lemma
	doc.Find("h2").First().Each(func(i int, s *goquery.Selection) {
		flag := s.Text()
		result = flag != lemma
	})

	d.cache.Set(language, lemma, result)
	return
}
