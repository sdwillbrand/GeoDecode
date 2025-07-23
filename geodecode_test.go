package geodecode_test

import (
	"log"
	"testing"

	geodecode "github.com/sdwillbrand/GeoDecode"
)

func TestFindLocation(t *testing.T) {
	// --- Test Case 1: Ocean coordinate (expects nearest land location) ---
	oceanCoord := [2]float64{0.0, 0.0} // Near Ghana
	log.Printf("Running test for ocean coordinate %v", oceanCoord)
	oceanLocation := geodecode.FindLocation(oceanCoord, true)

	if oceanLocation == nil {
		t.Fatalf("Expected to find a location for ocean coordinate %v, but got nil", oceanCoord)
	}
	// Verify it found a plausible nearby location.
	// You might need to adjust expected values based on your specific dataset.
	expectedOceanName := "Takoradi" // Or another city closest to 0,0 in your data
	expectedOceanCC := "GH"

	if oceanLocation.City != expectedOceanName {
		t.Errorf("For ocean coordinate %v: Expected name %q, got %q", oceanCoord, expectedOceanName, oceanLocation.City)
	}
	if oceanLocation.CC != expectedOceanCC {
		t.Errorf("For ocean coordinate %v: Expected CC %q, got %q", oceanCoord, expectedOceanCC, oceanLocation.CC)
	}
	log.Printf("Found location for ocean coordinate %v: %+v", oceanCoord, oceanLocation)

	// --- Test Case 2: Valid coordinate for a known city ---
	anadyrCoord := [2]float64{64.73424, 177.5103} // Anadyr, Russia
	log.Printf("Running test for known city coordinate %v", anadyrCoord)
	anadyrLocation := geodecode.FindLocation(anadyrCoord, true)

	if anadyrLocation == nil {
		t.Fatalf("Expected to find a location for Anadyr coordinate %v, but got nil", anadyrCoord)
	}
	// Assert specific details for Anadyr
	expectedAnadyrName := "Anadyr"
	expectedAnadyrCC := "RU"
	if anadyrLocation.City != expectedAnadyrName {
		t.Errorf("For Anadyr coordinate %v: Expected name %q, got %q", anadyrCoord, expectedAnadyrName, anadyrLocation.City)
	}
	if anadyrLocation.CC != expectedAnadyrCC {
		t.Errorf("For Anadyr coordinate %v: Expected CC %q, got %q", anadyrCoord, expectedAnadyrCC, anadyrLocation.CC)
	}
	log.Printf("Found location for known city coordinate %v: %+v", anadyrCoord, anadyrLocation)

	// --- Test Case 3: A truly "invalid" coordinate (out of bounds) ---
	// This should return nil, as your parser filters these.
	invalidCoord := [2]float64{999.0, 999.0} // Completely out of bounds
	log.Printf("Running test for truly invalid coordinate %v", invalidCoord)
	invalidLocation := geodecode.FindLocation(invalidCoord, true)

	if invalidLocation != nil {
		t.Errorf("Expected nil for truly invalid coordinate %v, but got %+v", invalidCoord, invalidLocation)
	}
	log.Printf("Confirmed nil for truly invalid coordinate %v", invalidCoord)
}
