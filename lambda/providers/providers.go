package providers

import (
	"time"
)

const (
	kmPerMi = 1.609344
)

type Location struct {
	Type          string     `json:"type"`
	Address       string     `json:"address"`
	Lat           float64    `json:"lat"`
	Lng           float64    `json:"lng"`
	DistanceMiles float64    `json:"distance_miles"`
	IsOpen        bool       `json:"is_open"`
	OpenTime      *time.Time `json:"open_time"`
	CloseTime     *time.Time `json:"close_time"`
}

type Provider interface {
	GetLocations(lat float64, lng float64, radiusMi float64, maxResults int) ([]Location, error)
}

var AllProviders = []string{
	TypeMcDonalds,
}
