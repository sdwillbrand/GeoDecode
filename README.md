# GeoDecode

![GitHub Release](https://img.shields.io/github/v/release/sdwillbrand/GeoDecode)
[![Go Reference](https://pkg.go.dev/badge/github.com/sdwillbrand/GeoDecode.svg)](https://pkg.go.dev/github.com/sdwillbrand/GeoDecode)

GeoDecode is a Go package that provides offline reverse geocoding. It allows you to find the nearest city and country for a given latitude and longitude coordinate.

## What it Does

This package takes a geographic coordinate (latitude and longitude) and returns the name of the closest known city and its country code. It's designed for quick lookups using an embedded dataset.
Features

- Offline Reverse Geocoding: Works without an internet connection after initial setup.

- City and Country Lookup: Provides the name of the nearest city and its ISO Alpha-2 country code.

- Fast Lookups: Uses a KD-Tree for efficient nearest neighbor searches on a large dataset.

- Embedded Data: The necessary geographic data is bundled directly into the package.

## Installation

To use GeoDecode in your Go project, you can install it using go get:

```bash
go get github.com/sdwillbrand/GeoDecode
```

## Usage

Here's a simple example of how to use GeoDecode in your Go application:

```go
package main

import (
  "fmt"
  "github.com/sdwillbrand/GeoDecode" // Import the GeoDecode package
)

func main() {
  // Define a coordinate (e.g., Berlin, Germany)
  coords := [2]float64{52.5200, 13.4050} // [lat, lng]

  // Find the nearest location.
  // The 'true' argument enables verbose logging during the initial data load.
  location := geodecode.FindLocation(coords, true)

  if location != nil {
    fmt.Printf("Found Location:\n")
    fmt.Printf("  City: %s\n", location.City)
    fmt.Printf("  Country Code: %s\n", location.CC)
    fmt.Printf("  Latitude: %.5f, Longitude: %.5f\n", location.Lat, location.Lon)
  } else {
    fmt.Println("Location not found for the given coordinates.")
  }

   // Example with an ocean coordinate (will return the nearest land location)
   oceanCoords := [2]float64{0.0, 0.0}
   oceanLocation := geodecode.FindLocation(oceanCoords, false) // No verbose logging for subsequent calls

  if oceanLocation != nil {
    fmt.Printf("\nClosest to ocean (0,0):\n")
    fmt.Printf("  City: %s\n", oceanLocation.City)
    fmt.Printf("  Country Code: %s\n", oceanLocation.CC)
  }
}
```

## Data Source

The geographic data used by GeoDecode is sourced from [rg_cities1000.csv](rg_cities1000.csv). This CSV file contains a list of cities with their coordinates and administrative information. The file is embedded directly into the Go package for ease of use.

## Contributing

If you find issues or have suggestions, please open an issue on the GitHub repository.

## License

GeoDecode is free and open source project licensed under the [MIT License](LICENSE.md).
