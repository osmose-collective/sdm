package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/brianloveswords/airtable"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jsonp"
	"github.com/go-chi/render"
)

const (
	BASE_ID    = "appyNzgElfcTGWF0V"
	TABLE_NAME = "Onboarding v2"
)

type LatLong struct {
	I string
	O struct {
		Status           string
		FormattedAddress string `json:"formattedAddress"`
		Lat              float64
		Lng              float64
	}
	E int
}

type OnboardingRecord struct {
	airtable.Record
	Fields struct {
		LatLong  string `json:"@map"`
		Location string `json:"@endroitpourinitierunecommune"`
		Amount   int    `json:"@nombredepersonnepourinitiercommune"`
		Comment  string `json:"@Commentaire"`
	}
	Translated struct {
		LatLong LatLong
	}
}

type OnboardingWithLocation struct {
	Lat      float64
	Long     float64
	Amount   int
	Location string
}

type Cache struct {
	Onboardings []*OnboardingRecord
}

var (
	cache Cache
	mutex sync.Mutex
)

func refreshCache() error {
	client := airtable.Client{
		APIKey: os.Getenv("AIRTABLE_TOKEN"),
		BaseID: BASE_ID,
	}
	table := client.Table(TABLE_NAME)
	entries := []OnboardingRecord{}
	table.List(&entries, &airtable.Options{
		Fields: []string{"LatLong", "Location", "Amount", "Comment"},
	})
	mutex.Lock()
	defer mutex.Unlock()
	cache.Onboardings = make([]*OnboardingRecord, 0)
	for _, entryRaw := range entries {
		entry := entryRaw
		_, i := utf8.DecodeRuneInString(entry.Fields.LatLong)
		trimmed := strings.TrimSpace(entry.Fields.LatLong[i:])
		dec, err := base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			log.Printf("failed to b64 decode: %v", err)
			continue
		}
		if err := json.Unmarshal(dec, &entry.Translated.LatLong); err != nil {
			log.Printf("failed to parse json: %v", err)
			continue
		}
		cache.Onboardings = append(cache.Onboardings, &entry)
	}
	return nil
}

func main() {
	if os.Getenv("GEN_AND_STOP") == "1" {
		if err := refreshCache(); err != nil {
			panic(err)
		}

		onboardings, err := getOnboardingsWithLocation()
		if err != nil {
			panic(err)
		}

		out, _ := json.Marshal(onboardings)
		if err := ioutil.WriteFile("./static/carte/data.js", append([]byte("var data = "), out...), 0644); err != nil {
			panic(err)
		}
		return
	}

	go func() {
		for {
			if err := refreshCache(); err != nil {
				log.Printf("failed to refresh cache: %v", err)
			}
			time.Sleep(time.Second * 10)
		}
	}()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(jsonp.Handler)

	r.Use(middleware.Timeout(5 * time.Second))

	r.Get("/onboardings.json", func(w http.ResponseWriter, r *http.Request) {
		onboardings, err := getOnboardingsWithLocation()
		if err != nil {
			log.Printf("failed to process onboardings: %v", err)
			return
		}
		render.JSON(w, r, onboardings)
	})

	fmt.Println("Listening on :3333...")
	panic(http.ListenAndServe("0.0.0.0:3333", r))
}

func getOnboardingsWithLocation() ([]OnboardingWithLocation, error) {
	mutex.Lock()
	defer mutex.Unlock()
	ret := []OnboardingWithLocation{}
	for _, entry := range cache.Onboardings {
		if entry.Translated.LatLong.O.Status == "ZERO_RESULTS" {
			continue
		}
		ret = append(ret, OnboardingWithLocation{
			Lat:      entry.Translated.LatLong.O.Lat,
			Long:     entry.Translated.LatLong.O.Lng,
			Location: entry.Translated.LatLong.O.FormattedAddress,
			Amount:   entry.Fields.Amount,
		})
	}
	return ret, nil
}
