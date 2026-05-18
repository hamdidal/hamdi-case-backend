package product

import (
	"testing"

	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── validateMaterials ────────────────────────────────────────────────────────

func TestValidateMaterials(t *testing.T) {
	tests := []struct {
		name    string
		mats    []matIn
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty slice is valid",
			mats:    []matIn{},
			wantErr: false,
		},
		{
			name: "exactly 100% passes",
			mats: []matIn{
				{Name: "Cotton", Percentage: 60},
				{Name: "Polyester", Percentage: 40},
			},
			wantErr: false,
		},
		{
			name: "within floating-point tolerance 99.999% passes",
			mats: []matIn{
				{Name: "Cotton", Percentage: 33.333},
				{Name: "Polyester", Percentage: 33.333},
				{Name: "Elastan", Percentage: 33.334},
			},
			wantErr: false,
		},
		{
			name: "single 100% material passes",
			mats: []matIn{{Name: "Merino Wool", Percentage: 100}},
			wantErr: false,
		},
		{
			name: "sum 99.9% is rejected",
			mats: []matIn{
				{Name: "Cotton", Percentage: 60},
				{Name: "Polyester", Percentage: 39.9},
			},
			wantErr: true,
			errMsg:  "material percentages must sum to exactly 100%",
		},
		{
			name: "sum 105% is rejected",
			mats: []matIn{
				{Name: "Cotton", Percentage: 60},
				{Name: "Polyester", Percentage: 45},
			},
			wantErr: true,
			errMsg:  "material percentages must sum to exactly 100%",
		},
		{
			name: "sum 0% is rejected",
			mats: []matIn{
				{Name: "Cotton", Percentage: 0},
				{Name: "Polyester", Percentage: 0},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMaterials(tc.mats)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── deepDiff ─────────────────────────────────────────────────────────────────

func TestDeepDiff_NoChanges(t *testing.T) {
	s := fullSnap{Name: "Hoodie", Brand: "EcoWear", Status: "draft"}
	changes := deepDiff(s, s)
	assert.Empty(t, changes, "identical snapshots should produce zero field changes")
}

func TestDeepDiff_ScalarFields(t *testing.T) {
	before := fullSnap{Name: "Old Name", Brand: "Brand A", Category: "ceket", Status: "draft"}
	after := fullSnap{Name: "New Name", Brand: "Brand A", Category: "pantolon", Status: "published"}

	changes := deepDiff(before, after)

	// Build a map for easy lookup by field name.
	byField := make(map[string]fieldChange)
	for _, fc := range changes {
		byField[fc.Field] = fc
	}

	require.Contains(t, byField, "name")
	assert.Equal(t, "Old Name", byField["name"].Before)
	assert.Equal(t, "New Name", byField["name"].After)

	require.Contains(t, byField, "category")
	assert.Equal(t, "ceket", byField["category"].Before)
	assert.Equal(t, "pantolon", byField["category"].After)

	require.Contains(t, byField, "status")
	assert.NotContains(t, byField, "brand", "unchanged field must not appear in diff")
}

func TestDeepDiff_MaterialAdded(t *testing.T) {
	before := fullSnap{
		Name:      "Shirt",
		Materials: []matSnap{{Name: "Cotton", Percentage: 100}},
	}
	after := fullSnap{
		Name: "Shirt",
		Materials: []matSnap{
			{Name: "Cotton", Percentage: 70},
			{Name: "Polyester", Percentage: 30},
		},
	}

	changes := deepDiff(before, after)
	byField := make(map[string]fieldChange)
	for _, fc := range changes {
		byField[fc.Field] = fc
	}

	// materials[0].percentage changed from 100 → 70
	require.Contains(t, byField, "materials[0].percentage")
	assert.Equal(t, float64(100), byField["materials[0].percentage"].Before)
	assert.Equal(t, float64(70), byField["materials[0].percentage"].After)

	// materials[1] is a new entry (Before must be nil)
	require.Contains(t, byField, "materials[1]")
	assert.Nil(t, byField["materials[1]"].Before)
	assert.NotNil(t, byField["materials[1]"].After)
}

func TestDeepDiff_MaterialRemoved(t *testing.T) {
	before := fullSnap{
		Name: "Jacket",
		Materials: []matSnap{
			{Name: "Wool", Percentage: 80},
			{Name: "Nylon", Percentage: 20},
		},
	}
	after := fullSnap{
		Name:      "Jacket",
		Materials: []matSnap{{Name: "Wool", Percentage: 100}},
	}

	changes := deepDiff(before, after)
	byField := make(map[string]fieldChange)
	for _, fc := range changes {
		byField[fc.Field] = fc
	}

	// materials[1] was removed (After must be nil)
	require.Contains(t, byField, "materials[1]")
	assert.NotNil(t, byField["materials[1]"].Before)
	assert.Nil(t, byField["materials[1]"].After)
}

func TestDeepDiff_CareAdded(t *testing.T) {
	before := fullSnap{Name: "Trousers"}
	after := fullSnap{
		Name: "Trousers",
		Care: &careSnap{WashTemperature: "30°C", DryClean: false},
	}

	changes := deepDiff(before, after)
	require.Len(t, changes, 1)
	assert.Equal(t, "care", changes[0].Field)
	assert.Nil(t, changes[0].Before)
	assert.NotNil(t, changes[0].After)
}

func TestDeepDiff_CareRemoved(t *testing.T) {
	before := fullSnap{
		Name: "Dress",
		Care: &careSnap{WashTemperature: "40°C"},
	}
	after := fullSnap{Name: "Dress"}

	changes := deepDiff(before, after)
	require.Len(t, changes, 1)
	assert.Equal(t, "care", changes[0].Field)
	assert.NotNil(t, changes[0].Before)
	assert.Nil(t, changes[0].After)
}

func TestDeepDiff_CareFieldChanged(t *testing.T) {
	before := fullSnap{
		Name: "Sweater",
		Care: &careSnap{WashTemperature: "30°C", Ironing: "low", DryClean: false, Bleaching: false},
	}
	after := fullSnap{
		Name: "Sweater",
		Care: &careSnap{WashTemperature: "40°C", Ironing: "low", DryClean: true, Bleaching: false},
	}

	changes := deepDiff(before, after)
	byField := make(map[string]fieldChange)
	for _, fc := range changes {
		byField[fc.Field] = fc
	}

	require.Contains(t, byField, "care.washTemperature")
	assert.Equal(t, "30°C", byField["care.washTemperature"].Before)
	assert.Equal(t, "40°C", byField["care.washTemperature"].After)

	require.Contains(t, byField, "care.dryClean")
	assert.Equal(t, false, byField["care.dryClean"].Before)
	assert.Equal(t, true, byField["care.dryClean"].After)

	assert.NotContains(t, byField, "care.ironing", "unchanged care field must not appear")
}

// ─── fullSnapFromProduct ──────────────────────────────────────────────────────

func TestFullSnapFromProduct(t *testing.T) {
	p := models.Product{
		Name:           "Test Product",
		Brand:          "BrandX",
		Category:       "t-shirt",
		Country:        "Turkey",
		ProductionDate: "2025-01-01",
		Status:         "published",
		Materials: []models.Material{
			{Name: "Cotton", Percentage: 90, Recycled: false},
			{Name: "Elastan", Percentage: 10, Recycled: true},
		},
		Care: &models.ProductCare{
			WashTemperature: "40°C",
			Ironing:         "medium",
			DryClean:        false,
			Bleaching:       false,
			Notes:           "gentle cycle",
		},
	}

	snap := fullSnapFromProduct(p)

	assert.Equal(t, "Test Product", snap.Name)
	assert.Equal(t, "BrandX", snap.Brand)
	require.Len(t, snap.Materials, 2)
	assert.Equal(t, "Cotton", snap.Materials[0].Name)
	assert.Equal(t, float64(90), snap.Materials[0].Percentage)
	assert.False(t, snap.Materials[0].Recycled)
	assert.True(t, snap.Materials[1].Recycled)
	require.NotNil(t, snap.Care)
	assert.Equal(t, "40°C", snap.Care.WashTemperature)
	assert.Equal(t, "gentle cycle", snap.Care.Notes)
}

func TestFullSnapFromProduct_NilCare(t *testing.T) {
	p := models.Product{Name: "Simple", Care: nil}
	snap := fullSnapFromProduct(p)
	assert.Nil(t, snap.Care)
}
