package main

import (
	"testing"
)

func TestValidation(t *testing.T) {
	body := `{}`
	req, err := validateRequest(body)

	if err == nil {
		t.Error("Expected errors")
	}

	body = `{"lat": 500, "lng": true}`
	req, err = validateRequest(body)

	if err == nil {
		t.Error("Expected errors")
	}

	body = `{"lat": 1.0, "lng": -2.0, "types": ["mcdonalds"]}`
	req, err = validateRequest(body)

	if err != nil {
		t.Errorf("Expected no errors; Got: %v", err)
	}

	if req.Lat != 1.0 || req.Lng != -2.0 {
		t.Error("Missing explicit values")
	}

	if req.DistanceMiles != 5.0 || req.MaxResults != 30 {
		t.Error("Missing implicit values")
	}
}
