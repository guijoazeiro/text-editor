package testutil

import (
	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewMockDB() (*gorm.DB, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	return openGORM(db, mock)
}

func NewMockDBWithQueryMatch() (*gorm.DB, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.NewWithDSN("sqlmock_db_0", sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		return nil, nil, err
	}
	return openGORM(db, mock)
}

func openGORM(db *sql.DB, mock sqlmock.Sqlmock) (*gorm.DB, sqlmock.Sqlmock, error) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, nil, err
	}
	return gormDB, mock, nil
}
