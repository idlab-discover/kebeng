package database

import (
	"fmt"
    "time"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	gormPostgres "gorm.io/driver/postgres"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source
)


func NewDatabase() (*gorm.DB, error) {
    return  createDatabaseWithDSN(getDSN())
} 

func createDatabaseWithDSN(connectionString string) (*gorm.DB, error) {
    var db *gorm.DB
    var err error

    maxRetries := 10
    retryInterval := 3 * time.Second

    for try := 0; try < maxRetries; try++ {
        db, err = gorm.Open(gormPostgres.Open(connectionString), &gorm.Config{})
        if err == nil {
            logrus.Info("Connected to database")

            err = runMigrations(db)
            if err != nil {
                logrus.Errorf("Migration failed: %v", err)
            }
            return db, nil
        }
        logrus.Errorf("Failed to connect to database at try %d: %v",try, err)
        time.Sleep(retryInterval)
    }
    logrus.Errorf("Failed to connect to database after %d retries", maxRetries)
    return nil, err
}

func getDSN() string {
    return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable TimeZone=UTC",
        viper.GetString("database.host"),
        viper.GetInt("database.port"),
        viper.GetString("database.user"),
        viper.GetString("database.dbname"),
        viper.GetString("database.password"),
    )
}

// runs the migration files in the /migrations folder
func runMigrations(db *gorm.DB) error {
    logrus.Info("Running database migrations")

    sqlDB, err := db.DB()
    if err != nil {
        return fmt.Errorf("failed to get raw database connection: %v", err)
    }
    
    driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %v", err)
    }
    
    // the path here is the path in the container where the migration files are stored
    m, err := migrate.NewWithDatabaseInstance(
        "file:///app/migrations",
        "postgres",
        driver,
    )
    if err != nil {   
        return fmt.Errorf("failed to initialize migrate: %v", err)
    }

    err = m.Up()
    if err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %v", err)
    }

    return nil
}



