package product

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

//go:embed fonts/DejaVuSansCondensed.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSansCondensed-Bold.ttf
var fontBold []byte

//go:embed fonts/DejaVuSansCondensed-Oblique.ttf
var fontItalic []byte

const (
	brandR, brandG, brandB = 45, 106, 79   // #2D6A4F — Emerald Green
	zebraR, zebraG, zebraB = 236, 246, 241  // #ECF6F1 — light brand tint
	headerR, headerG, headerB = 240, 248, 244 // very pale green for section bg
)

func GetProductPDF(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	err = database.DB.Preload("Materials").Preload("Care").First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve product"})
		return
	}

	lang := c.DefaultQuery("lang", "en")
	pdf := buildPassportPDF(p, lang)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate PDF"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="passport-%s.pdf"`, p.ID))
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}

func newPDF() *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)
	pdf.AddUTF8FontFromBytes("DejaVu", "I", fontItalic)
	return pdf
}

type pdfLabels struct {
	basicInfo    string
	brand        string
	category     string
	country      string
	productionDate string
	materials    string
	material     string
	percentage   string
	recycled     string
	noMaterials  string
	care         string
	fieldLabel   string
	fieldValue   string
	noCare       string
	wash         string
	iron         string
	dryCleaning  string
	bleaching    string
	notes        string
	yes          string
	no           string
	statusDraft  string
	statusPublished string
}

func labelsFor(lang string) pdfLabels {
	if lang == "tr" {
		return pdfLabels{
			basicInfo:       "Temel Bilgiler",
			brand:           "Marka",
			category:        "Kategori",
			country:         "Üretim Ülkesi",
			productionDate:  "Üretim Tarihi",
			materials:       "Malzeme Kompozisyonu",
			material:        "Malzeme",
			percentage:      "Yüzde (%)",
			recycled:        "Geri Dönüştürülmüş",
			noMaterials:     "Malzeme kaydedilmemiş.",
			care:            "Bakım Talimatları",
			fieldLabel:      "Alan",
			fieldValue:      "Değer",
			noCare:          "Bakım talimatı kaydedilmemiş.",
			wash:            "Yıkama Sıcaklığı",
			iron:            "Ütüleme",
			dryCleaning:     "Kuru Temizleme",
			bleaching:       "Ağartma",
			notes:           "Notlar",
			yes:             "Evet",
			no:              "Hayır",
			statusDraft:     "Taslak",
			statusPublished: "Yayında",
		}
	}
	return pdfLabels{
		basicInfo:       "Basic Information",
		brand:           "Brand",
		category:        "Category",
		country:         "Country of Origin",
		productionDate:  "Production Date",
		materials:       "Material Composition",
		material:        "Material",
		percentage:      "Percentage (%)",
		recycled:        "Recycled",
		noMaterials:     "No materials recorded.",
		care:            "Care Instructions",
		fieldLabel:      "Field",
		fieldValue:      "Value",
		noCare:          "No care instructions recorded.",
		wash:            "Wash Temperature",
		iron:            "Ironing",
		dryCleaning:     "Dry Cleaning",
		bleaching:       "Bleaching",
		notes:           "Notes",
		yes:             "Yes",
		no:              "No",
		statusDraft:     "Draft",
		statusPublished: "Published",
	}
}

func buildPassportPDF(p models.Product, lang string) *gofpdf.Fpdf {
	pdf := newPDF()
	pdf.SetMargins(22, 22, 22)
	pdf.AddPage()

	lbl := labelsFor(lang)
	pageW, _ := pdf.GetPageSize()
	contentW := pageW - 44

	// ── Header banner ────────────────────────────────────────────
	pdf.SetFillColor(brandR, brandG, brandB)
	pdf.Rect(0, 0, pageW, 52, "F")

	// Brand mark (small white square with inner texture)
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(22, 14, 6, 6, "F")
	pdf.SetFillColor(zebraR, zebraG, zebraB)
	pdf.Rect(23, 15, 4, 4, "F")

	// Brand name
	pdf.SetFont("DejaVu", "B", 11)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(31, 12)
	pdf.CellFormat(60, 7, "Kobe", "", 0, "L", false, 0, "")

	// Tagline
	pdf.SetFont("DejaVu", "", 7)
	pdf.SetTextColor(180, 220, 200)
	pdf.SetXY(31, 19)
	pdf.CellFormat(60, 5, "DPP MANAGEMENT", "", 0, "L", false, 0, "")

	// Product name — right side of header
	pdf.SetFont("DejaVu", "B", 17)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(22, 10)
	pdf.CellFormat(contentW, 13, p.Name, "", 1, "R", false, 0, "")

	// Status badge
	statusLabel := lbl.statusDraft
	if p.Status == "published" {
		statusLabel = lbl.statusPublished
	}
	pdf.SetFont("DejaVu", "", 8)
	pdf.SetTextColor(180, 220, 200)
	pdf.SetXY(22, 25)
	pdf.CellFormat(contentW, 6, statusLabel, "", 1, "R", false, 0, "")

	// Passport ID
	pdf.SetFont("DejaVu", "", 7)
	pdf.SetTextColor(140, 200, 170)
	pdf.SetXY(22, 40)
	pdf.CellFormat(contentW, 5, fmt.Sprintf("ID: %s", p.ID), "", 1, "R", false, 0, "")

	pdf.SetY(64)

	// ── Basic Information ────────────────────────────────────────
	sectionHeader(pdf, contentW, lbl.basicInfo)

	infoRows := [][]string{
		{lbl.brand, strOrNA(p.Brand)},
		{lbl.category, strOrNA(p.Category)},
		{lbl.country, strOrNA(p.Country)},
		{lbl.productionDate, strOrNA(p.ProductionDate)},
	}
	renderInfoRows(pdf, contentW, infoRows)
	pdf.Ln(10)

	// ── Material Composition ─────────────────────────────────────
	sectionHeader(pdf, contentW, lbl.materials)

	if len(p.Materials) == 0 {
		pdf.SetFont("DejaVu", "I", 9)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(contentW, 9, lbl.noMaterials, "", 1, "L", false, 0, "")
	} else {
		colW := []float64{contentW * 0.50, contentW * 0.25, contentW * 0.25}
		tableHeader(pdf, colW, []string{lbl.material, lbl.percentage, lbl.recycled})
		for i, m := range p.Materials {
			recycledLabel := lbl.no
			if m.Recycled {
				recycledLabel = lbl.yes
			}
			if i%2 == 0 {
				pdf.SetFillColor(zebraR, zebraG, zebraB)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.SetFont("DejaVu", "", 9)
			pdf.SetTextColor(35, 35, 35)
			pdf.CellFormat(colW[0], 9, "  "+m.Name, "LRB", 0, "L", true, 0, "")
			pdf.CellFormat(colW[1], 9, fmt.Sprintf("%.1f%%", m.Percentage), "RB", 0, "C", true, 0, "")
			pdf.CellFormat(colW[2], 9, recycledLabel, "RB", 1, "C", true, 0, "")
		}
	}
	pdf.Ln(10)

	// ── Care Instructions ────────────────────────────────────────
	sectionHeader(pdf, contentW, lbl.care)

	if p.Care == nil {
		pdf.SetFont("DejaVu", "I", 9)
		pdf.SetTextColor(160, 160, 160)
		pdf.CellFormat(contentW, 9, lbl.noCare, "", 1, "L", false, 0, "")
	} else {
		colW := []float64{contentW * 0.38, contentW * 0.62}
		tableHeader(pdf, colW, []string{lbl.fieldLabel, lbl.fieldValue})

		boolStr := func(b bool) string {
			if b {
				return lbl.yes
			}
			return lbl.no
		}

		careRows := [][]string{
			{lbl.wash, strOrNA(p.Care.WashTemperature)},
			{lbl.iron, strOrNA(p.Care.Ironing)},
			{lbl.dryCleaning, boolStr(p.Care.DryClean)},
			{lbl.bleaching, boolStr(p.Care.Bleaching)},
			{lbl.notes, strOrNA(p.Care.Notes)},
		}
		for i, row := range careRows {
			if i%2 == 0 {
				pdf.SetFillColor(zebraR, zebraG, zebraB)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.SetFont("DejaVu", "B", 9)
			pdf.SetTextColor(70, 70, 70)
			pdf.CellFormat(colW[0], 9, "  "+row[0], "LRB", 0, "L", true, 0, "")
			pdf.SetFont("DejaVu", "", 9)
			pdf.SetTextColor(35, 35, 35)
			pdf.CellFormat(colW[1], 9, "  "+row[1], "RB", 1, "L", true, 0, "")
		}
	}

	// ── Footer ───────────────────────────────────────────────────
	pdf.SetY(-20)
	pdf.SetDrawColor(brandR, brandG, brandB)
	pdf.SetLineWidth(0.4)
	pdf.Line(22, pdf.GetY(), pageW-22, pdf.GetY())
	pdf.Ln(4)
	pdf.SetFont("DejaVu", "B", 7)
	pdf.SetTextColor(brandR, brandG, brandB)
	pdf.CellFormat(contentW/2, 5, "Kobe · DPP MANAGEMENT", "", 0, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 7)
	pdf.SetTextColor(160, 160, 160)
	pdf.CellFormat(contentW/2, 5, fmt.Sprintf("Passport ID: %s", p.ID), "", 0, "R", false, 0, "")

	return pdf
}

func sectionHeader(pdf *gofpdf.Fpdf, w float64, title string) {
	// Accent bar
	pdf.SetFillColor(brandR, brandG, brandB)
	pdf.Rect(pdf.GetX(), pdf.GetY(), 3, 9, "F")

	// Section label background
	pdf.SetFillColor(headerR, headerG, headerB)
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(brandR, brandG, brandB)
	pdf.CellFormat(w, 9, "   "+title, "B", 1, "L", true, 0, "")

	pdf.SetDrawColor(zebraR, zebraG, zebraB)
	pdf.Ln(3)
}

func tableHeader(pdf *gofpdf.Fpdf, colWidths []float64, headers []string) {
	pdf.SetFont("DejaVu", "B", 8)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(brandR, brandG, brandB)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, "  "+h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

func renderInfoRows(pdf *gofpdf.Fpdf, contentW float64, rows [][]string) {
	labelW := contentW * 0.35
	valW := contentW * 0.65
	for i, row := range rows {
		if i%2 == 0 {
			pdf.SetFillColor(zebraR, zebraG, zebraB)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetFont("DejaVu", "B", 9)
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(labelW, 9, "  "+row[0], "LRB", 0, "L", true, 0, "")
		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(35, 35, 35)
		pdf.CellFormat(valW, 9, "  "+row[1], "RB", 1, "L", true, 0, "")
	}
}

func strOrNA(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
