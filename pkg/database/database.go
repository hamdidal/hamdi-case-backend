package database

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

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
}

func migrate() {
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Material{},
		&models.CareInstruction{},
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

func randomPassword(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}
