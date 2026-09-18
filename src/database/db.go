package db

import (
	"log/slog"

	"github.com/yazu-codes/scanme-analytics.git/src/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	admin         string
	adminPassword string
	DSN           string
	Connection    *gorm.DB
	logger        *slog.Logger
}

func NewDB(a, ap, dsn string, l *slog.Logger) *DB {
	return &DB{admin: a, adminPassword: ap, DSN: dsn, Connection: nil, logger: l}
}

func (d *DB) Connect() {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: d.DSN,
	}), &gorm.Config{})

	if err != nil {
		d.logger.Error(err.Error())
	}

	d.Connection = db
}

func (d *DB) AutoMigrate() {
	d.logger.Info("Auto-migrating database schema...")
	if err := d.Connection.AutoMigrate(
		&models.Event{},
	); err != nil {
		d.logger.Error(err.Error())
	}
}

func (d *DB) adminExists() bool {
	var count int64
	d.Connection.Table("users").Where("role = ?", "admin").Count(&count)
	return count > 0
}

func (d *DB) SeedAdmin() error {
	if d.adminExists() {
		d.logger.Info("Admin user already exists, skipping seeding.")
		return nil
	}

	// Hash password for admin123
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(d.adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (email, password, name, role)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (email) DO NOTHING
	`

	d.logger.Info(query)
	d.logger.Info(d.admin)
	d.logger.Info(d.adminPassword)

	result := d.Connection.Exec(query, d.admin, string(hashedPassword), "Admin", "admin")
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		d.logger.Info("✅ Admin user created")
	} else {
		d.logger.Info("Admin user already exists")
	}

	return nil
}
