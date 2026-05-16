package database

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	DB = db
	log.Println("database connection established")

	migrate()
	seedAdmin()
	seedUsers()
	seedProducts()
}

func migrate() {
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Material{},
		&models.ProductCare{},
		&models.CareInstruction{}, // legacy — kept to avoid schema drift
		&models.AuditLog{},
		&models.ProductVersion{},
	); err != nil {
		log.Fatalf("auto migration failed: %v", err)
	}
}

func seedAdmin() {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	var existing models.User
	if err := DB.Where("username = ?", username).First(&existing).Error; err == nil {
		return
	}

	password := randomPassword(16)
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash admin password: %v", err)
	}

	admin := models.User{
		Username: username,
		Password: string(hashed),
		Role:     models.RoleAdmin,
	}
	if err := DB.Create(&admin).Error; err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	log.Printf("=== ADMIN SEEDED === username: %s | password: %s ===", username, password)
}

// seedProducts inserts 15 representative textile products on first run.
// The check is idempotent — if any products exist the function returns early.
func seedProducts() {
	var count int64
	DB.Model(&models.Product{}).Count(&count)
	if count > 0 {
		return
	}

	type productSeed struct {
		name           string
		brand          string
		category       string
		country        string
		productionDate string
		status         string
		materials      []models.Material
		care           models.ProductCare
	}

	seeds := []productSeed{
		{
			name: "Organic Cotton Hoodie", brand: "EcoWear", category: "ceket",
			country: "Türkiye", productionDate: "2025-09-15", status: "published",
			materials: []models.Material{
				{Name: "Organic Cotton", Percentage: 85, Recycled: false, Origin: "India"},
				{Name: "Recycled Polyester", Percentage: 15, Recycled: true, Origin: "Taiwan"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Düşük Isı", DryClean: false, Bleaching: false, Notes: "İçi dışına çevirerek yıkayın."},
		},
		{
			name: "Recycled Denim Jacket", brand: "ReForm", category: "ceket",
			country: "İtalya", productionDate: "2025-07-01", status: "published",
			materials: []models.Material{
				{Name: "Recycled Denim", Percentage: 70, Recycled: true, Origin: "Italy"},
				{Name: "Organic Cotton", Percentage: 25, Recycled: false, Origin: "USA"},
				{Name: "Elastan", Percentage: 5, Recycled: false, Origin: "Germany"},
			},
			care: models.ProductCare{WashTemperature: "40°C", Ironing: "Uygun", DryClean: false, Bleaching: false},
		},
		{
			name: "Merino Wool Sweater", brand: "AlpinePure", category: "ic-giyim",
			country: "Avusturya", productionDate: "2025-11-20", status: "published",
			materials: []models.Material{
				{Name: "Merino Wool", Percentage: 100, Recycled: false, Origin: "New Zealand"},
			},
			care: models.ProductCare{WashTemperature: "El Yıkama", Ironing: "Uygun Değil", DryClean: true, Bleaching: false, Notes: "Düz bir zeminde kurutun."},
		},
		{
			name: "Hemp Linen Trousers", brand: "TerraThread", category: "pantolon",
			country: "Portekiz", productionDate: "2025-05-10", status: "published",
			materials: []models.Material{
				{Name: "Hemp", Percentage: 55, Recycled: false, Origin: "France"},
				{Name: "Linen", Percentage: 45, Recycled: false, Origin: "Belgium"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Düşük Isı", DryClean: false, Bleaching: false},
		},
		{
			name: "Bamboo Basics T-Shirt", brand: "SoftRoot", category: "t-shirt",
			country: "Çin", productionDate: "2026-01-08", status: "published",
			materials: []models.Material{
				{Name: "Bamboo Viscose", Percentage: 95, Recycled: false, Origin: "China"},
				{Name: "Elastan", Percentage: 5, Recycled: false, Origin: "Germany"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Düşük Isı", DryClean: false, Bleaching: false},
		},
		{
			name: "Recycled Fleece Pullover", brand: "LoopBack", category: "ic-giyim",
			country: "Türkiye", productionDate: "2025-10-05", status: "published",
			materials: []models.Material{
				{Name: "Recycled Polyester Fleece", Percentage: 100, Recycled: true, Origin: "Turkey"},
			},
			care: models.ProductCare{WashTemperature: "40°C", Ironing: "Uygun Değil", DryClean: false, Bleaching: false, Notes: "Mikrofiber torbada yıkayın."},
		},
		{
			name: "Fair Trade Cotton Chinos", brand: "CommonGround", category: "pantolon",
			country: "Bangladeş", productionDate: "2025-06-22", status: "published",
			materials: []models.Material{
				{Name: "Fair Trade Cotton", Percentage: 97, Recycled: false, Origin: "Bangladesh"},
				{Name: "Elastan", Percentage: 3, Recycled: false, Origin: "Germany"},
			},
			care: models.ProductCare{WashTemperature: "40°C", Ironing: "Uygun", DryClean: false, Bleaching: false},
		},
		{
			name: "Lyocell Shirt", brand: "BioWeave", category: "t-shirt",
			country: "Avusturya", productionDate: "2025-08-30", status: "published",
			materials: []models.Material{
				{Name: "TENCEL™ Lyocell", Percentage: 100, Recycled: false, Origin: "Austria"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Düşük Isı", DryClean: false, Bleaching: false},
		},
		{
			name: "Upcycled Wool Coat", brand: "ReForm", category: "ceket",
			country: "İngiltere", productionDate: "2025-12-01", status: "draft",
			materials: []models.Material{
				{Name: "Upcycled Wool", Percentage: 80, Recycled: true, Origin: "UK"},
				{Name: "Recycled Polyester Lining", Percentage: 20, Recycled: true, Origin: "Germany"},
			},
			care: models.ProductCare{WashTemperature: "El Yıkama", Ironing: "Düşük Isı", DryClean: true, Bleaching: false, Notes: "Yalnızca kuru temizleme önerilir."},
		},
		{
			name: "Corduroy Work Pants", brand: "CraftWear", category: "pantolon",
			country: "Portekiz", productionDate: "2025-04-14", status: "published",
			materials: []models.Material{
				{Name: "Organic Cotton", Percentage: 94, Recycled: false, Origin: "India"},
				{Name: "Elastan", Percentage: 6, Recycled: false, Origin: "Germany"},
			},
			care: models.ProductCare{WashTemperature: "40°C", Ironing: "Uygun", DryClean: false, Bleaching: false},
		},
		{
			name: "Linen Summer Dress", brand: "Soluna", category: "diger",
			country: "Fransa", productionDate: "2026-02-14", status: "published",
			materials: []models.Material{
				{Name: "European Linen", Percentage: 100, Recycled: false, Origin: "France"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Düşük Isı", DryClean: false, Bleaching: false, Notes: "Gölgede kurutun."},
		},
		{
			name: "Pique Cotton Polo", brand: "EcoWear", category: "t-shirt",
			country: "Türkiye", productionDate: "2025-05-28", status: "published",
			materials: []models.Material{
				{Name: "Organic Cotton", Percentage: 100, Recycled: false, Origin: "Turkey"},
			},
			care: models.ProductCare{WashTemperature: "40°C", Ironing: "Uygun", DryClean: false, Bleaching: false},
		},
		{
			name: "Recycled Nylon Windbreaker", brand: "AirCycle", category: "ceket",
			country: "Japonya", productionDate: "2025-09-02", status: "published",
			materials: []models.Material{
				{Name: "Recycled Nylon (ECONYL®)", Percentage: 100, Recycled: true, Origin: "Italy"},
			},
			care: models.ProductCare{WashTemperature: "30°C", Ironing: "Uygun Değil", DryClean: false, Bleaching: false},
		},
		{
			name: "Thermal Underwear Set", brand: "AlpinePure", category: "ic-giyim",
			country: "Avusturya", productionDate: "2025-11-11", status: "published",
			materials: []models.Material{
				{Name: "Merino Wool", Percentage: 70, Recycled: false, Origin: "New Zealand"},
				{Name: "Recycled Polyester", Percentage: 30, Recycled: true, Origin: "Taiwan"},
			},
			care: models.ProductCare{WashTemperature: "El Yıkama", Ironing: "Uygun Değil", DryClean: false, Bleaching: false},
		},
		{
			name: "Deadstock Silk Blouse", brand: "Remnant", category: "diger",
			country: "İtalya", productionDate: "2026-03-01", status: "draft",
			materials: []models.Material{
				{Name: "Deadstock Silk", Percentage: 100, Recycled: true, Origin: "Italy"},
			},
			care: models.ProductCare{WashTemperature: "El Yıkama", Ironing: "Düşük Isı", DryClean: true, Bleaching: false, Notes: "Yalnızca soğuk suyla yıkayın."},
		},
	}

	adminID := uuid.New() // fallback for audit logs during seeding
	var admin models.User
	if err := DB.Where("role = ?", models.RoleAdmin).First(&admin).Error; err == nil {
		adminID = admin.ID
	}

	now := time.Now()
	for i, s := range seeds {
		sku := fmt.Sprintf("SKU-%04d", i+1)
		// Spread creation dates over the past 12 months for realistic data
		createdAt := now.AddDate(0, -(len(seeds)-1-i), 0).Add(time.Duration(rand.Intn(20)) * 24 * time.Hour)
		p := models.Product{
			Name:           s.name,
			Brand:          s.brand,
			Category:       s.category,
			Country:        s.country,
			ProductionDate: s.productionDate,
			Status:         s.status,
			SKU:            sku,
			CreatedBy:      admin.Username,
			CreatedAt:      createdAt,
		}
		for _, m := range s.materials {
			p.Materials = append(p.Materials, models.Material{
				Name:       m.Name,
				Percentage: m.Percentage,
				Recycled:   m.Recycled,
				Origin:     m.Origin,
			})
		}
		care := s.care
		p.Care = &care

		if err := DB.Create(&p).Error; err != nil {
			log.Printf("seed product %q failed: %v", s.name, err)
			continue
		}

		// Write an audit log for the seeded product
		raw := fmt.Sprintf(`{"seeded":true,"name":%q}`, s.name)
		auditEntry := models.AuditLog{
			UserID:     adminID,
			Username:   admin.Username,
			Action:     "create",
			EntityType: "product",
			EntityID:   p.ID,
			EntityName: s.name,
			Changes:    models.JSONB(raw),
		}
		DB.Create(&auditEntry)
	}

	log.Printf("seeded %d products", len(seeds))
}

// seedUsers creates 30 auditor users for pagination testing if none exist yet.
func seedUsers() {
	var count int64
	DB.Model(&models.User{}).Where("role = ?", models.RoleAuditor).Count(&count)
	if count > 0 {
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()

	firstNames := []string{
		"alice", "bob", "carol", "dave", "eve", "frank", "grace", "hank",
		"iris", "jack", "kate", "leo", "mia", "noah", "olivia", "peter",
		"quinn", "rose", "sam", "tina", "uma", "victor", "wendy", "xander",
		"yara", "zoe", "aaron", "bella", "carlos", "diana",
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte("Auditor123!"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seedUsers: failed to hash password: %v", err)
		return
	}

	for i, name := range firstNames {
		// Spread creation dates over the past 18 months
		daysBack := rng.Intn(540) + 1
		createdAt := now.AddDate(0, 0, -daysBack)

		u := models.User{
			Username:  fmt.Sprintf("%s.auditor%02d", name, i+1),
			Password:  string(hashed),
			Role:      models.RoleAuditor,
			CreatedAt: createdAt,
		}
		if err := DB.Create(&u).Error; err != nil {
			log.Printf("seedUsers: failed to create user %q: %v", u.Username, err)
		}
	}

	log.Printf("seeded %d auditor users", len(firstNames))
}

func randomPassword(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}
