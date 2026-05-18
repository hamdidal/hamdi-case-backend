package product

import (
	"fmt"

	"github.com/hamdidal/dpp-backend/internal/models"
)

type matSnap struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Recycled   bool    `json:"recycled"`
}

type careSnap struct {
	WashTemperature string `json:"washTemperature"`
	Ironing         string `json:"ironing"`
	DryClean        bool   `json:"dryClean"`
	Bleaching       bool   `json:"bleaching"`
	Notes           string `json:"notes"`
}

// fullSnap is a complete, auditable snapshot of a product including nested associations.
type fullSnap struct {
	Name           string    `json:"name"`
	Brand          string    `json:"brand"`
	Category       string    `json:"category"`
	Country        string    `json:"country"`
	ProductionDate string    `json:"productionDate"`
	Status         string    `json:"status"`
	Materials      []matSnap `json:"materials"`
	Care           *careSnap `json:"care,omitempty"`
}

// fieldChange records a single field-level difference between two product states.
type fieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// fullSnapFromProduct captures a complete snapshot from a loaded Product (associations must be preloaded).
func fullSnapFromProduct(p models.Product) fullSnap {
	mats := make([]matSnap, len(p.Materials))
	for i, m := range p.Materials {
		mats[i] = matSnap{Name: m.Name, Percentage: m.Percentage, Recycled: m.Recycled}
	}
	s := fullSnap{
		Name:           p.Name,
		Brand:          p.Brand,
		Category:       p.Category,
		Country:        p.Country,
		ProductionDate: p.ProductionDate,
		Status:         p.Status,
		Materials:      mats,
	}
	if p.Care != nil {
		s.Care = &careSnap{
			WashTemperature: p.Care.WashTemperature,
			Ironing:         p.Care.Ironing,
			DryClean:        p.Care.DryClean,
			Bleaching:       p.Care.Bleaching,
			Notes:           p.Care.Notes,
		}
	}
	return s
}

// fullSnapFromReq builds a fullSnap from an incoming update request.
func fullSnapFromReq(req productRequest, status string) fullSnap {
	mats := make([]matSnap, len(req.Materials))
	for i, m := range req.Materials {
		mats[i] = matSnap{Name: m.Name, Percentage: m.Percentage, Recycled: m.Recycled}
	}
	s := fullSnap{
		Name:           req.Name,
		Brand:          req.Brand,
		Category:       req.Category,
		Country:        req.Country,
		ProductionDate: req.ProductionDate,
		Status:         status,
		Materials:      mats,
	}
	if req.CareInstructions != nil {
		s.Care = &careSnap{
			WashTemperature: req.CareInstructions.WashTemperature,
			Ironing:         req.CareInstructions.Ironing,
			DryClean:        req.CareInstructions.DryClean,
			Bleaching:       req.CareInstructions.Bleaching,
			Notes:           req.CareInstructions.Notes,
		}
	}
	return s
}

// deepDiff returns a flat list of field-level changes between two product snapshots.
// Materials are compared by index position. Care fields are compared individually.
func deepDiff(before, after fullSnap) []fieldChange {
	var changes []fieldChange

	addStr := func(field, b, a string) {
		if b != a {
			changes = append(changes, fieldChange{Field: field, Before: b, After: a})
		}
	}

	addStr("name", before.Name, after.Name)
	addStr("brand", before.Brand, after.Brand)
	addStr("category", before.Category, after.Category)
	addStr("country", before.Country, after.Country)
	addStr("productionDate", before.ProductionDate, after.ProductionDate)
	addStr("status", before.Status, after.Status)

	// Materials: compare by index
	maxLen := len(before.Materials)
	if len(after.Materials) > maxLen {
		maxLen = len(after.Materials)
	}
	for i := 0; i < maxLen; i++ {
		prefix := fmt.Sprintf("materials[%d]", i)
		if i >= len(before.Materials) {
			changes = append(changes, fieldChange{Field: prefix, Before: nil, After: after.Materials[i]})
		} else if i >= len(after.Materials) {
			changes = append(changes, fieldChange{Field: prefix, Before: before.Materials[i], After: nil})
		} else {
			b, a := before.Materials[i], after.Materials[i]
			addStr(prefix+".name", b.Name, a.Name)
			if b.Percentage != a.Percentage {
				changes = append(changes, fieldChange{Field: prefix + ".percentage", Before: b.Percentage, After: a.Percentage})
			}
			if b.Recycled != a.Recycled {
				changes = append(changes, fieldChange{Field: prefix + ".recycled", Before: b.Recycled, After: a.Recycled})
			}
		}
	}

	// Care instructions
	switch {
	case before.Care == nil && after.Care != nil:
		changes = append(changes, fieldChange{Field: "care", Before: nil, After: *after.Care})
	case before.Care != nil && after.Care == nil:
		changes = append(changes, fieldChange{Field: "care", Before: *before.Care, After: nil})
	case before.Care != nil && after.Care != nil:
		b, a := before.Care, after.Care
		addStr("care.washTemperature", b.WashTemperature, a.WashTemperature)
		addStr("care.ironing", b.Ironing, a.Ironing)
		if b.DryClean != a.DryClean {
			changes = append(changes, fieldChange{Field: "care.dryClean", Before: b.DryClean, After: a.DryClean})
		}
		if b.Bleaching != a.Bleaching {
			changes = append(changes, fieldChange{Field: "care.bleaching", Before: b.Bleaching, After: a.Bleaching})
		}
		addStr("care.notes", b.Notes, a.Notes)
	}

	return changes
}
