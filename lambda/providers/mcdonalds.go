package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bradfitz/latlong"
	"github.com/kellydunn/golang-geo"
)

const TypeMcDonalds = "mcdonalds"

const urlTemplate = "https://www.mcdonalds.com/googleapps/GoogleRestaurantLocAction.do?method=searchLocation&latitude={{.Lat}}&longitude={{.Lng}}&radius={{.RadiusKm}}&maxResults={{.MaxResults}}&country=us&language=en-us"

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func round(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}

type params struct {
	Lat        float64
	Lng        float64
	RadiusKm   float64
	MaxResults int
}

type mcDonalds struct {
	template *template.Template
}

func NewMcDonalds() Provider {
	tmpl, err := template.New("urlTemplate").Parse(urlTemplate)
	if err != nil {
		panic(err)
	}

	return &mcDonalds{
		template: tmpl,
	}
}

func parseTime(timeStr string, now time.Time) time.Time {
	parts := strings.Split(timeStr, ":")
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])

	return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
}

func (m *mcDonalds) filterResults(body []byte, from *geo.Point) ([]Location, error) {
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	features, ok := data["features"].([]interface{})
	if !ok {
		return nil, errors.New("Missing data from API")
	}

	var locs []Location
	for _, f := range features {
		loc := f.(map[string]interface{})

		props := loc["properties"].(map[string]interface{})
		filterType := props["filterType"].([]interface{})
		driveThru := false
		for _, t := range filterType {
			if t.(string) == "DRIVETHRU" {
				driveThru = true
				break
			}
		}

		if !driveThru {
			continue
		}

		geom := loc["geometry"].(map[string]interface{})
		coords := geom["coordinates"].([]interface{})
		lat := coords[1].(float64)
		lng := coords[0].(float64)
		distanceM := from.GreatCircleDistance(geo.NewPoint(lat, lng))

		address := strings.Title(strings.ToLower(props["addressLine1"].(string)))

		tz := latlong.LookupZoneName(lat, lng)
		tLoc, _ := time.LoadLocation(tz)
		now := time.Now().In(tLoc)

		hours := strings.Split(props["driveTodayHours"].(string), " - ")
		twentyFourHours := hours[0] == hours[1]

		open := parseTime(hours[0], now)
		close := parseTime(hours[1], now)
		if close.Hour() < open.Hour() {
			close = close.Add(time.Hour * 24)
		}

		var openTime, closeTime *time.Time
		if !twentyFourHours {
			openTime = &open
			closeTime = &close
		}

		locs = append(locs, Location{
			Type:          TypeMcDonalds,
			Address:       address,
			Lat:           lat,
			Lng:           lng,
			DistanceMiles: round(distanceM/kmPerMi, 2),
			IsOpen:        twentyFourHours || (now.After(open) && now.Before(close)),
			OpenTime:      openTime,
			CloseTime:     closeTime,
		})
	}

	sort.Slice(locs, func(i, j int) bool {
		return locs[i].DistanceMiles < locs[j].DistanceMiles
	})

	return locs, nil
}

func (m *mcDonalds) GetLocations(lat float64, lng float64, radiusMi float64, maxResults int) ([]Location, error) {
	params := params{
		Lat:        lat,
		Lng:        lng,
		RadiusKm:   radiusMi * kmPerMi,
		MaxResults: maxResults,
	}

	var buf bytes.Buffer
	m.template.Execute(&buf, params)
	url := buf.String()

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	locs, err := m.filterResults(body, geo.NewPoint(lat, lng))
	if err != nil {
		return nil, err
	}

	return locs[:min(len(locs), maxResults)], nil
}
