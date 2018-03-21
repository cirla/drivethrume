package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cirla/drivethrume/lambda/providers"
	"github.com/xeipuuv/gojsonschema"
)

var requestSchemaJSON = fmt.Sprintf(`{
	"$schema": "http://json-schema.org/draft-04/schema#",
	"type": "object",
	"required": [ "lat", "lng" ],
	"properties": {
		"lat": {
			"type": "number",
			"minimum": -90.0,
			"maximum": 90.0
		},
		"lng": {
			"type": "number",
			"minimum": -180.0,
			"maximum": 180.0
		},
		"distance_miles": {
			"type": "number",
			"minimum": 0.0,
			"exclusiveMinimum": true,
			"maximum": 25.0
		},
		"max_results": {
			"type": "integer",
			"minimum": 1,
			"maximum": 30
		},
		"show_closed": {
			"type": "boolean"
		},
		"types": {
			"type": "array",
			"items": {
				"type": "string",
				"enum": ["%s"]
			},
			"uniqueItems": true
		}
	}
}`, strings.Join(providers.AllProviders, `", "`))

type Request struct {
	Lat           float64  `json:"lat"`
	Lng           float64  `json:"lng"`
	DistanceMiles float64  `json:"distance_miles"`
	MaxResults    int      `json:"max_results"`
	ShowClosed    bool     `json:"show_closed"`
	Types         []string `json:"types"`
}

type Response struct {
	Locations []providers.Location `json:"locations"`
	Errors    []string             `json:"errors,omitempty"`
}

var providerMap map[string]providers.Provider
var requestSchema *gojsonschema.Schema

func init() {
	providerMap = map[string]providers.Provider{
		providers.TypeMcDonalds: providers.NewMcDonalds(),
	}

	schemaLoader := gojsonschema.NewStringLoader(requestSchemaJSON)

	var err error
	requestSchema, err = gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		log.Fatalf("Error initializing request schema: %v", err)
	}
}

func validateRequest(body string) (Request, error) {
	var req Request

	bodyLoader := gojsonschema.NewStringLoader(body)
	result, err := requestSchema.Validate(bodyLoader)
	if err != nil {
		return req, err
	}

	if !result.Valid() {
		errStrings := make([]string, 0, len(result.Errors()))
		for _, e := range result.Errors() {
			errStrings = append(errStrings, e.String())
		}
		return req, fmt.Errorf("[%s]", strings.Join(errStrings, ", "))
	}

	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return req, err
	}

	if req.DistanceMiles == 0.0 {
		req.DistanceMiles = 5.0
	}

	if req.MaxResults == 0 {
		req.MaxResults = 30
	}

	if req.Types == nil || len(req.Types) == 0 {
		req.Types = make([]string, 0, len(providerMap))
		for k := range providerMap {
			req.Types = append(req.Types, k)
		}
	}

	return req, nil
}

func handler(gwRequest events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	request, err := validateRequest(gwRequest.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}, err
	}

	locations := []providers.Location{}
	var errs []string

	for _, pType := range request.Types {
		provider := providerMap[pType]
		pLocs, err := provider.GetLocations(request.Lat, request.Lng, request.DistanceMiles, request.MaxResults)
		if err == nil {
			locations = append(locations, pLocs...)
		} else {
			errs = append(errs, err.Error())
		}
	}

	response := Response{
		Locations: locations,
		Errors:    errs,
	}

	json, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, errors.New("unable to marshal response")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func main() {
	lambda.Start(handler)
}
