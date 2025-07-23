package main

import (
	"fmt"

	geodecode "github.com/sdwillbrand/GeoDecode"
)

func main() {
	fmt.Println("Geocoder instantiated, but data not loaded yet.")

	fmt.Println("\nTesting single coordinate through FindLocation()...")
	city1 := [2]float64{37.78674, -122.39222} // Near San Francisco
	result1 := geodecode.FindLocation(city1, true)
	if result1 != nil {
		fmt.Printf("Result for %v: %+v\n", city1, *result1)
	} else {
		fmt.Printf("Result for %v: Not found\n", city1)
	}

	fmt.Println("\nTesting another single coordinate (data already loaded)...")
	city2 := [2]float64{48.8566, 2.3522}            // Paris
	result2 := geodecode.FindLocation(city2, false) // Verbose=False as data is loaded
	if result2 != nil {
		fmt.Printf("Result for %v: %+v\n", city2, *result2)
	} else {
		fmt.Printf("Result for %v: Not found\n", city2)
	}

	fmt.Println("\nTesting multiple coordinates through RGeocoder.Query()...")
	coordsList := [][2]float64{
		{52.5200, 13.4050},   // Berlin
		{40.7128, -74.0060},  // New York City
		{-33.8688, 151.2093}, // Sydney
	}
	geocoderInstance := geodecode.GetRGeocoder(false) // Gets the existing singleton instance
	resultsList := geocoderInstance.Query(coordsList...)
	fmt.Println("Results for multiple coordinates:")
	for i, coord := range coordsList {
		if i < len(resultsList) {
			fmt.Printf("  %v: %+v\n", coord, resultsList[i])
		} else {
			fmt.Printf("  %v: Not found\n", coord)
		}
	}

	fmt.Println("\nTesting edge case: Coordinate in ocean...")
	oceanCoord := [2]float64{0.0, 0.0} // Middle of the ocean
	resultOcean := geodecode.FindLocation(oceanCoord, false)
	if resultOcean != nil {
		fmt.Printf("Result for %v: %+v\n", oceanCoord, *resultOcean)
	} else {
		fmt.Printf("Result for %v: Not found\n", oceanCoord)
	}

	fmt.Println("\nTesting edge case: Empty list input to query...")
	emptyResults := geocoderInstance.Query()
	fmt.Printf("Result for empty list query: %+v\n", emptyResults)
}
