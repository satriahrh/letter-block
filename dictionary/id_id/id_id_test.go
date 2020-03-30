package id_id_test

import (
	"github.com/satriahrh/letter-block/dictionary/id_id"

	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type DataDictionary struct {
	mock.Mock
}

func (d *DataDictionary) Get(lang, key string) (bool, bool) {
	args := d.Called(lang, key)
	return args.Bool(0), args.Bool(1)
}

func (d *DataDictionary) Set(lang, key string, value bool) {
	d.Called(lang, key, value)
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

type RoundTripErrorFunc func(req *http.Request) *http.Response

func (f RoundTripErrorFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected error")
}

func TestIdId_LemmaIsValid(t *testing.T) {
	language := "id-id"
	lemma := "word"

	testSuite := func(httpClient *http.Client) (dataDictionary *DataDictionary, idId *id_id.IdId) {
		dataDictionary = &DataDictionary{}
		idId = id_id.NewIdId(
			dataDictionary,
			httpClient,
		)
		return
	}

	t.Run("ExistOnCache", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			dataDictionary, idId := testSuite(nil)

			dataDictionary.
				On("Get", "id-id", "word").
				Return(true, true)

			result, err := idId.LemmaIsValid(lemma)
			if !assert.NoError(t, err, "using cache") {
				t.FailNow()
			}

			assert.True(t, result, "valid as expected in mock")
		})

		t.Run("Valid", func(t *testing.T) {
			dataDictionary, idId := testSuite(nil)

			dataDictionary.
				On("Get", language, lemma).
				Return(false, true)

			result, err := idId.LemmaIsValid(lemma)
			if !assert.NoError(t, err, "using cache") {
				t.FailNow()
			}

			assert.False(t, result, "valid as expected in mock")
		})
	})
	t.Run("ErrorRequestToKBBI", func(t *testing.T) {
		t.Run("WhenRequesting", func(t *testing.T) {
			client := &http.Client{
				Transport: RoundTripErrorFunc(func(req *http.Request) *http.Response {
					return nil
				}),
			}

			dataDictionary, idId := testSuite(client)

			dataDictionary.
				On("Get", "id-id", "word").
				Return(false, false)

			_, err := idId.LemmaIsValid(lemma)
			assert.Regexp(t, "unexpected error", err.Error(), "unexpected error")
		})
		t.Run("GotNon200Response", func(t *testing.T) {
			client := &http.Client{
				Transport: RoundTripFunc(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 500,
						Body:       ioutil.NopCloser(bytes.NewBufferString(`internal server error`)),
					}
				}),
			}

			dataDictionary, idId := testSuite(client)

			dataDictionary.
				On("Get", "id-id", "word").
				Return(false, false)

			_, err := idId.LemmaIsValid(lemma)
			assert.EqualError(t, err, id_id.ErrorHttpUnexpected.Error(), "500 error")
		})
	})

}
