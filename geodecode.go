// Package geodecode provides functionality to find the nearest
// geographical location (city, town) for a given latitude and longitude
// coordinate from a pre-loaded dataset. It utilizes a KD-Tree for efficient
// nearest neighbor searches.
package geodecode

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/biter777/countries"
	"gonum.org/v1/gonum/spatial/kdtree"
)

//go:embed rg_cities1000.csv
var rawCSVData []byte

const (
	rgFilename = "rg_cities1000.csv"
)

// Location represents a geographical point with associated administrative data.
type Location struct {
	Lat     float64 // Latitude of the location.
	Lon     float64 // Longitude of the location.
	City    string  // Name of the location (e.g., city name).
	Admin1  string  // First-level administrative division (e.g., state, province).
	Admin2  string  // Second-level administrative division (e.g., county, region).
	CC      string  // Country Code (e.g., US, GB).
	Country string  // Name of the country
}

// geoPoint wraps a Location and satisfies kdtree.Comparable
type geoPoint struct {
	LatLon [2]float64
	Index  int // Store the original index to retrieve the full Location data
}

// Compare returns the signed distance of p from the plane passing through
// c and perpendicular to the dimension d.
func (p geoPoint) Compare(c kdtree.Comparable, d kdtree.Dim) float64 {
	q := c.(geoPoint)
	return p.LatLon[d] - q.LatLon[d] // d is kdtree.Dim, which correctly indexes [2]float64
}

// Dims returns the number of dimensions described by the receiver (2 for Lat/Lon).
func (p geoPoint) Dims() int {
	return 2
}

// Distance returns the squared Euclidean distance between c and the receiver.
func (p geoPoint) Distance(c kdtree.Comparable) float64 {
	q := c.(geoPoint)
	dLat := p.LatLon[0] - q.LatLon[0]
	dLon := p.LatLon[1] - q.LatLon[1]
	return dLat*dLat + dLon*dLon
}

// geoPoints implements kdtree.Interface AND sort.Interface for a slice of geoPoint
type geoPoints []geoPoint

// Len returns the length of the list.
func (p geoPoints) Len() int {
	return len(p)
}

// Index returns the ith element of the list of points.
func (p geoPoints) Index(i int) kdtree.Comparable {
	return p[i]
}

// Swap swaps the elements at indices i and j.
func (p geoPoints) Swap(i, j int) {
	if i < 0 || j < 0 || i >= p.Len() || j >= p.Len() {
		return
	}
	p[i], p[j] = p[j], p[i]
}

// currentSortDim is a package-level variable used by Less to know which dimension to sort by.
var currentSortDim kdtree.Dim

// Less reports whether the element at index i should sort before the element at index j.
func (p geoPoints) Less(i, j int) bool {
	// Explicitly convert kdtree.Dim to int for array indexing
	return p[i].LatLon[int(currentSortDim)] < p[j].LatLon[int(currentSortDim)]
}

// Pivot partitions the list based on the dimension specified.
func (p geoPoints) Pivot(dim kdtree.Dim) int {
	currentSortDim = dim // Set the package-level variable
	// It's important that Partition handles the base cases (len <= 1)
	// gracefully without trying to access out-of-bounds indices.
	// If Gonum's Partition itself panics with 1 element, it might be a library bug,
	// or we need to ensure we don't call it with less than 2 elements from our side.
	return kdtree.Partition(p, int(dim))
}

// Slice returns a slice of the list using zero-based half-open indexing.
func (p geoPoints) Slice(start, end int) kdtree.Interface {
	return p[start:end]
}

// RGeocoder represents the main reverse geocoding service.
// It holds the KD-Tree and the loaded location data.
type RGeocoder struct {
	tree      *kdtree.Tree
	locations []Location // Store original Location structs, indexed by geoPoint.Index
	once      sync.Once
	verbose   bool
}

var (
	geocoderInstance *RGeocoder
	geocoderOnce     sync.Once
)

// GetRGeocoder returns a singleton instance of the reverse geocoder.
// The geocoder's data is loaded and the KD-Tree is built only once,
// on the first call to this function.
// The 'verbose' parameter controls whether detailed loading and warning messages
// are printed to the console.
func GetRGeocoder(verbose bool) *RGeocoder {
	geocoderOnce.Do(func() {
		geocoderInstance = &RGeocoder{
			verbose: verbose,
		}
	})
	geocoderInstance.verbose = verbose
	return geocoderInstance
}

// loadData loads the data from the embedded CSV and builds the KD-Tree.
func (rg *RGeocoder) loadData() {
	if rg.verbose {
		log.Println("geodecode: Loading and processing geodata...")
	}

	startTime := time.Now()

	var reader *csv.Reader
	if len(rawCSVData) > 0 {
		reader = csv.NewReader(bytes.NewReader(rawCSVData))
	} else {
		filePath := filepath.Join(".", rgFilename)
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("geodecode: Error: Data file '%s' not found: %v", filePath, err)
			return
		}
		defer file.Close()
		reader = csv.NewReader(file)
	}

	header, err := reader.Read()
	if err != nil {
		log.Printf("geodecode: Error reading CSV header: %v", err)
		return
	}

	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	requiredCols := []string{"lat", "lon", "city", "admin1", "admin2", "cc"}
	for _, reqCol := range requiredCols {
		if _, ok := colMap[reqCol]; !ok {
			log.Printf("geodecode: Error: CSV file missing required column: %s", reqCol)
			return
		}
	}

	var parsedGeoPoints geoPoints  // This will hold our kdtree.Comparable points
	var loadedLocations []Location // This will hold the full Location data

	for i := 0; ; i++ { // Start from 0 for index, CSV row number starts at 1 (after header)
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("geodecode: Warning: Skipping row %d due to read error: %v", i+1, err)
			continue
		}

		latStr := record[colMap["lat"]]
		lonStr := record[colMap["lon"]]

		lat, errLat := strconv.ParseFloat(latStr, 64)
		lon, errLon := strconv.ParseFloat(lonStr, 64)

		if errLat != nil || errLon != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			if rg.verbose {
				log.Printf("geodecode: Warning: Skipping row %d with invalid coordinates: lat='%s', lon='%s', Error: %v, %v", i+1, latStr, lonStr, errLat, errLon)
			}
			continue
		}

		// Store the full location data
		fullLocation := Location{
			Lat:    lat,
			Lon:    lon,
			City:   record[colMap["city"]],
			Admin1: record[colMap["admin1"]],
			Admin2: record[colMap["admin2"]],
			CC:     record[colMap["cc"]],
		}
		loadedLocations = append(loadedLocations, fullLocation)

		// Create the geoPoint for the KD-Tree, linking back to the original index
		parsedGeoPoints = append(parsedGeoPoints, geoPoint{
			LatLon: [2]float64{lat, lon},
			Index:  len(loadedLocations) - 1, // Index in the loadedLocations slice
		})

	}

	if len(parsedGeoPoints) == 0 {
		log.Println("geodecode: Warning: No valid coordinates loaded.")
		return
	}
	if rg.verbose {
		log.Printf("geodecode: Successfully parsed %d valid points from CSV.", len(parsedGeoPoints))
	}

	if len(parsedGeoPoints) == 1 {
		log.Println("geodecode: Only one valid coordinate loaded. KDTree will not be built.")
		rg.locations = loadedLocations
		rg.tree = nil
		return
	}

	// Build the KD-Tree
	rg.tree = kdtree.New(parsedGeoPoints, false) // `false` for no bounding (not strictly needed for nearest neighbor)
	rg.locations = loadedLocations               // Store the full location data

	if rg.verbose {
		endTime := time.Now()
		log.Printf("geodecode: Data loaded, KDTree built in %.2f seconds. %d locations indexed.",
			endTime.Sub(startTime).Seconds(), len(rg.locations))
	}
}

// Query finds the nearest location to the given coordinate.
// It returns a Location struct if found, otherwise an empty Location{}.
// It also performs validation on the input coordinate.
func (rg *RGeocoder) Query(coordinates ...[2]float64) []Location {
	rg.once.Do(rg.loadData) // Ensure data is loaded lazily

	if rg.tree == nil && len(rg.locations) == 0 { // Check if data loading failed or was empty
		return []Location{}
	}

	if len(coordinates) == 0 {
		return []Location{}
	}

	results := make([]Location, 0, len(coordinates))

	for _, coord := range coordinates {
		// Handle case where only one location was loaded and no KDTree was built
		lat := coord[0]
		lon := coord[1]

		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			if rg.verbose {
				log.Printf("geodecode: Invalid query coordinate received: Lat=%.4f, Lon=%.4f. Returning empty location.", lat, lon)
			}
			return nil
		}
		if rg.tree == nil && len(rg.locations) == 1 {
			// If there's only one location, that must be the nearest.
			results = append(results, rg.locations[0])
			continue
		}

		queryPoint := geoPoint{LatLon: coord} // Create a geoPoint for querying

		// Use the KD-Tree's Nearest method
		nearestComparable, distSq := rg.tree.Nearest(queryPoint)

		if nearestComparable == nil || math.IsInf(distSq, 1) {
			// No nearest point found (e.g., empty tree)
			if rg.verbose {
				log.Printf("geodecode: Warning: No nearest point found for %v", coord)
			}
			results = append(results, Location{}) // Append an empty Location for consistency
			continue
		}

		nearestGeoPoint, ok := nearestComparable.(geoPoint)
		if !ok {
			// This should not happen if our implementation is correct
			log.Printf("geodecode: Error: KDTree returned a non-geoPoint type.")
			results = append(results, Location{})
			continue
		}

		// Retrieve the full Location data using the stored index
		if nearestGeoPoint.Index >= 0 && nearestGeoPoint.Index < len(rg.locations) {
			results = append(results, rg.locations[nearestGeoPoint.Index])
		} else {
			log.Printf("geodecode: Error: KDTree returned invalid index %d", nearestGeoPoint.Index)
			results = append(results, Location{})
		}
	}

	return results
}

// FindLocation is a convenience function to query the geocoder directly
// for a single coordinate.
// It returns a pointer to the nearest Location found, or nil if no location
// is found (e.g., input coordinate is invalid or no data is loaded).
// The 'verbose' parameter controls logging for the internal geocoder instance.
//
// coordinate: [lat, lng]
//
// Example usage:
//
//	location := geodecode.FindLocation([2]float64{34.0522, -118.2437}, false) // Los Angeles
//	if location != nil {
//	    fmt.Printf("City: %s, Country: %s\n", location.City, location.Country)
//	}
func FindLocation(coordinate [2]float64, verbose bool) *Location {
	if coordinate[0] < -90 || coordinate[0] > 90 || coordinate[1] < -180 || coordinate[1] > 180 {
		// If the coordinate itself is outside valid bounds, return nil
		return nil
	}
	geocoder := GetRGeocoder(verbose)
	results := geocoder.Query(coordinate)
	if len(results) > 0 {
		result := &results[0]
		country := countries.ByName(result.CC)
		result.Country = country.Info().Name
		return &results[0]
	}
	return nil
}
