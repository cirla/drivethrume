package providers

import (
	"io/ioutil"
	"testing"

	"github.com/kellydunn/golang-geo"
)

func TestMcDonalds(t *testing.T) {
	m := &mcDonalds{}

	body, _ := ioutil.ReadFile("./mcdonalds_test.json")

	locs, err := m.filterResults(body, geo.NewPoint(40.8768, -73.3246))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(locs) < 4 {
		t.Errorf("Fewer than expected locs: %d", len(locs))
	}
}
