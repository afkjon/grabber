package crawlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	db "github.com/afkjon/grabber/internal/database"
	"github.com/afkjon/grabber/internal/model"
)

// Location represents a row in your 'locations' table
type Location struct {
	ID              int
	Address         string
	Latitude        sql.NullFloat64 // Use sql.NullFloat64 for nullable columns
	Longitude       sql.NullFloat64
	LocationType    sql.NullString  // Use sql.NullString for nullable columns
	FullAPIResponse json.RawMessage // To store the raw JSONB data
	Geocoded        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Global variable for API key (consider context for larger apps)
var googleAPIKey string
var client *http.Client

// Rate limiter for API calls (e.g., 100 requests per second is common for Google)
var apiLimiter <-chan time.Time

func GeocodeAddresses() error {
	// Initialize HTTP client
	client = &http.Client{Timeout: 10 * time.Second}

	// Read API key from environment variable
	googleAPIKey = os.Getenv("GOOGLE_MAPS_GEOCODING_API_KEY")
	if googleAPIKey == "" {
		fmt.Println("Warning: GOOGLE_MAPS_GEOCODING_API_KEY environment variable not set.")
		fmt.Println("Please set it before running the program.")
		os.Exit(1)
	}

	// Initialize rate limiter: 100 requests per second (1 request every 10ms)
	// You might adjust this based on your Google Maps API quota and rate limits.
	// For production, consider using a library like 'go.uber.org/ratelimit' for more robust control.
	rateLimitPerSecond := 10
	apiLimiter = time.Tick(time.Second / time.Duration(rateLimitPerSecond))

	err := db.Connect()
	if err != nil {
		fmt.Printf("Error opening database connection: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Fetch ungeocoded locations
	rows, err := db.GetShopsPendingGeocoding()
	if err != nil {
		fmt.Printf("Error querying locations: %v\n", err)
		return err
	}

	// Print the addresses to be geocoded
	for _, shops := range rows {
		// Wait for the rate limiter
		<-apiLimiter
		// Process each location
		fmt.Printf("Shops ID: %d, Address: %s\n", shops.ID, shops.Address)
		geoData, fullResponse, err := geocodeAddress(shops.Address)
		if err != nil {
			fmt.Printf("Failed to geocode address '%s' (ID: %d): %v\n", shops.Address, shops.ID, err)
			continue
		}

		if geoData == nil {
			fmt.Printf("No geocoding results found for address '%s' (ID: %d). Skipping update.\n", shops.Address, shops.ID)
			continue
		}

		fmt.Printf("Geocoded address '%s' (ID: %d): Lat: %.6f, Lng: %.6f, Type: %s\n",
			shops.Address, shops.ID, geoData.Lat, geoData.Lng, geoData.LocationType)

		err = db.UpdateLocation(shops.ID, geoData, fullResponse)
		if err != nil {
			fmt.Printf("Failed to update location ID %d: %v\n", shops.ID, err)
		} else {
			fmt.Printf("Successfully geocoded and updated ID: %d (Lat: %.6f, Lng: %.6f, Type: %s)\n",
				shops.ID, geoData.Lat, geoData.Lng, geoData.LocationType)
		}
	}

	fmt.Println("Geocoding process completed.")
	return nil
}

// geocodeAddress calls the Google Maps Geocoding API
func geocodeAddress(address string) (*model.GoogleGeocodeResponseResultLocation, json.RawMessage, error) {
	if googleAPIKey == "" {
		return nil, nil, fmt.Errorf("Google Maps API Key is not set")
	}

	encodedAddress := url.QueryEscape(address)
	apiUrl := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", encodedAddress, googleAPIKey)

	resp, err := client.Get(apiUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read API response body: %w", err)
	}

	var geoResponse model.GoogleGeocodeResponse
	if err := json.Unmarshal(body, &geoResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if geoResponse.Status != "OK" {
		return nil, nil, fmt.Errorf("API returned status: %s (Body: %s)", geoResponse.Status, string(body))
	}

	if len(geoResponse.Results) == 0 {
		return nil, body, nil // No results, but status was OK
	}

	// Always pick the first result (you can add more sophisticated logic here)
	firstResult := geoResponse.Results[0]
	location := firstResult.Geometry.Location
	locationType := firstResult.Geometry.LocationType

	geoData := &model.GoogleGeocodeResponseResultLocation{
		Lat:          location.Lat,
		Lng:          location.Lng,
		LocationType: locationType,
	}

	return geoData, body, nil
}
