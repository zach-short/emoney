package manager

import (
	"strings"
	"testing"
)

// ExtractPropertyDetails is the only function on the property-management path
// that touches no database, so it is the only one testable end to end here.
// HandleHouseManagement and HandlePropertySaleMortgage both open with
// config.DB.Collection, which is a nil *mongo.Database in a test binary.

func TestExtractPropertyDetailsRejectsNonArray(t *testing.T) {
	_, err := ExtractPropertyDetails(map[string]any{"propertyId": "abc"})

	if err == nil || err.Error() != "properties field is not an array" {
		t.Fatalf("expected \"properties field is not an array\", got %v", err)
	}
}

func TestExtractPropertyDetailsRejectsNonObjectItem(t *testing.T) {
	_, err := ExtractPropertyDetails([]any{"abc"})

	if err == nil || err.Error() != "property item is not a valid object" {
		t.Fatalf("expected \"property item is not a valid object\", got %v", err)
	}
}

func TestExtractPropertyDetailsRejectsMissingPropertyID(t *testing.T) {
	_, err := ExtractPropertyDetails([]any{map[string]any{"count": float64(3)}})

	if err == nil || !strings.Contains(err.Error(), "missing or invalid propertyId") {
		t.Fatalf("expected a missing-propertyId error, got %v", err)
	}
}

func TestExtractPropertyDetailsRejectsNonStringPropertyID(t *testing.T) {
	_, err := ExtractPropertyDetails([]any{map[string]any{"propertyId": float64(42)}})

	if err == nil || !strings.Contains(err.Error(), "missing or invalid propertyId") {
		t.Fatalf("expected a missing-propertyId error, got %v", err)
	}
}

func TestExtractPropertyDetailsReturnsNilForEmptyArray(t *testing.T) {
	details, err := ExtractPropertyDetails([]any{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(details) != 0 {
		t.Fatalf("expected no property details, got %d", len(details))
	}
}

func TestExtractPropertyDetailsPreservesOrder(t *testing.T) {
	details, err := ExtractPropertyDetails([]any{
		map[string]any{"propertyId": "first", "count": float64(1)},
		map[string]any{"propertyId": "second", "count": float64(2)},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(details) != 2 {
		t.Fatalf("expected 2 property details, got %d", len(details))
	}
	if details[0].PropertyID != "first" || details[0].Count != 1 {
		t.Fatalf("expected {first 1} at index 0, got %+v", details[0])
	}
	if details[1].PropertyID != "second" || details[1].Count != 2 {
		t.Fatalf("expected {second 2} at index 1, got %+v", details[1])
	}
}

func TestExtractPropertyDetailsTruncatesFractionalCount(t *testing.T) {
	details, err := ExtractPropertyDetails([]any{
		map[string]any{"propertyId": "abc", "count": 2.9},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if details[0].Count != 2 {
		t.Fatalf("expected a count of 2, got %d", details[0].Count)
	}
}

func TestExtractPropertyDetailsDefaultsMissingCountToZero(t *testing.T) {
	// Documents current behaviour, which is not obviously right: a HOUSES
	// operation writes Count straight into developmentLevel
	// (manager/propertyManager.go:25), so a payload that omits count razes
	// every house on the property instead of being rejected. Harmless for
	// MORTGAGE/UNMORTGAGE/SELL, which ignore Count. Raised in HANDOFF 4.
	details, err := ExtractPropertyDetails([]any{
		map[string]any{"propertyId": "abc"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if details[0].Count != 0 {
		t.Fatalf("expected a count of 0, got %d", details[0].Count)
	}
}

func TestExtractPropertyDetailsDefaultsNonNumericCountToZero(t *testing.T) {
	// Same defaulting, reached a second way: encoding/json produces float64 for
	// every number, so a count sent as a JSON string misses the float64 arm.
	details, err := ExtractPropertyDetails([]any{
		map[string]any{"propertyId": "abc", "count": "3"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if details[0].Count != 0 {
		t.Fatalf("expected a count of 0, got %d", details[0].Count)
	}
}
