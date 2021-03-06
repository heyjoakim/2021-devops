package services

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/heyjoakim/devops-21/helpers"
	"github.com/heyjoakim/devops-21/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DbContext defines the application
type DbContext struct {
	db *gorm.DB
}

var (
	dsn         string
	environment string
	dbContext   DbContext
	lock        = &sync.Mutex{}
)

func configureEnv() {
	envFilePath := helpers.GetFullPath("../.env")
	err := godotenv.Load(envFilePath)
	if err != nil {
		log.Println("Error loading .env file - using system variables.")
	}

	environment = os.Getenv("ENVIRONMENT")
	dsn = os.Getenv("DB_CONNECTION")
}

func (d *DbContext) initialize() {
	configureEnv()
	db, err := d.connectDb()
	if err != nil {
		log.Panic(err)
	}
	d.db = db
}

func (d *DbContext) connectDb() (*gorm.DB, error) {
	fmt.Println(environment)
	if environment == "develop" {
		fmt.Println("Using local SQLite db")
		return gorm.Open(sqlite.Open("./tmp/minitwit.db"), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	} else if environment == "production" {
		fmt.Println("Using remote postgres db")
		return gorm.Open(postgres.New(
			postgres.Config{
				DSN:                  dsn,
				PreferSimpleProtocol: true, // disables implicit prepared statement usage
			}),
			&gorm.Config{
				NamingStrategy: schema.NamingStrategy{
					SingularTable: true,
				},
			})
	} else if environment == "testing" {
		fmt.Println("Using in memory SQLite db")

		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	}
	log.Panic("Environment is not specified in the .env file")
	return nil, nil
}

// initDb creates the database tables.
func (d *DbContext) initDb() {
	err := d.db.AutoMigrate(&models.User{}, &models.Follower{}, &models.Message{}, &models.Config{})

	if err != nil {
		log.Println("Migration error:", err)
	}
}

// GetDbInstance returns DbContext with specific environment db
func GetDbInstance() DbContext {
	if dbContext.db == nil {
		lock.Lock()
		defer lock.Unlock()
		if dbContext.db == nil {
			log.Println("Creating Single Instance Now")
			dbContext.initialize()
			dbContext.initDb() // AutoMigrate
		}
	}
	return dbContext

}
