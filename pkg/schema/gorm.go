package schema

import (
	"gorm.io/gorm"
)

// AllModels returns all schema models for GORM AutoMigrate.
func AllModels() []interface{} {
	return []interface{}{
		&DataSource{},
		&NameString{},
		&Canonical{},
		&CanonicalFull{},
		&CanonicalStem{},
		&NameStringIndex{},
		&Word{},
		&WordNameString{},
		&VernacularString{},
		&VernacularStringIndex{},
	}
}

// Migrate runs GORM AutoMigrate to create or update schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
