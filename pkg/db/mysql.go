package db

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitMySQLAndMigrate(dsn string, migrateModels ...interface{}) *gorm.DB {
	parts := strings.Split(dsn, "/")
	if len(parts) >= 2 {
		dbName := strings.Split(parts[1], "?")[0]
		instanceDSN := parts[0] + "/"
		tmpDB, err := gorm.Open(mysql.Open(instanceDSN), &gorm.Config{})
		if err == nil {
			createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName)
			tmpDB.Exec(createSQL)
			sqlDB, _ := tmpDB.DB()
			sqlDB.Close()
		}
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if len(migrateModels) > 0 {
		for _, model := range migrateModels {
			if err := db.AutoMigrate(model); err != nil {
				// Multiple services may migrate the same table concurrently at startup.
				// Ignore MySQL race errors like:
				// 1050 table already exists, 1060 duplicate column.
				if isIgnorableMigrationError(err) {
					log.Printf("migration skipped for %s due to concurrent create race: %v", modelName(model), err)
					continue
				}
				log.Fatalf("failed to migrate %s: %v", modelName(model), err)
			}
		}
	}

	return db
}

func isIgnorableMigrationError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "error 1050") ||
		strings.Contains(errMsg, "error 1060") ||
		strings.Contains(errMsg, "error 1061") ||
		strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "duplicate column") ||
		strings.Contains(errMsg, "duplicate key name")
}

func modelName(model interface{}) string {
	if model == nil {
		return "<nil>"
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Name() != "" {
		return t.Name()
	}
	return t.String()
}
