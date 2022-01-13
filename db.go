package main

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "log"
    "time"
    "os"
)

type User struct {
    ID int
    Username string `gorm:"not null;unique"`
    Password string `gorm:"not null"`
    Friends []*User `gorm:"many2many:user_friends"`
}

const (
    DB_FILENAME = "mydb.db"
)

func startDB() *gorm.DB {
    newLogger := logger.New(
        log.New(os.Stdout, "\r\n", log.LstdFlags),
        logger.Config{
            SlowThreshold: time.Second,
            LogLevel: logger.Info,
            IgnoreRecordNotFoundError: false,
            Colorful: true,
        },
    )


    db, err := gorm.Open(sqlite.Open(DB_FILENAME), &gorm.Config{Logger: newLogger})
    if err != nil {
		log.Fatal(err)
	}

    db.AutoMigrate(&User{})
    return db
}
