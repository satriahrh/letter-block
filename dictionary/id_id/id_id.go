package id_id

import (
	"log"

	"github.com/satriahrh/letter-block/data"

	"errors"
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
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
	_, _ = d.cache.Get(language, lemma)
	if exist {
		return result, nil
	}

	// Request To KBBI
	url := fmt.Sprintf("%v/%v", baseUrl, lemma)
	log.Println(url)
	res, err := d.httpClient.Get(url)
	if err != nil {
		return
	}
	if res.StatusCode != 200 {
		err = ErrorHttpUnexpected
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
