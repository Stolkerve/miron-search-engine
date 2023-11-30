package db

import (
	"log"
	"os"
	"time"

	"github.com/Stolkerve.com/miron-search-engine/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const TF_IDF_QUERY = `
SELECT 
    (word_freqs.count / documents.words_count) * ? as tf_idf, 
    documents.url 
FROM word_freqs 
INNER JOIN documents ON word_freqs.document_id = documents.id 
WHERE word_freqs.word = ? 
GROUP BY word_freqs.count, documents.words_count, documents.url
`

var Pool *gorm.DB

func SetupDB() {
	dsn := os.Getenv("DSN")
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Error,           // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,                   // Don't include params in the SQL log
			Colorful:                  false,                  // Disable color
		},
	)
	if db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		Logger:                 newLogger,
	}); err != nil {
		log.Fatalln(err)
	} else {
		Pool = db
		if err := Pool.AutoMigrate(models.Document{}, models.WordFreq{}); err != nil {
			log.Fatalln(err)
		}
	}
}
