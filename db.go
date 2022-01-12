package main

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "log"
)

type User struct {
    ID int
    Username string `gorm:"not null;unique"`
    Password string `gorm:"not null"`
    Friends []*User
}

const (
    DB_FILENAME = "mydb.db"
)

func startDB() *gorm.DB {
    db, err := gorm.Open(sqlite.Open(DB_FILENAME), &gorm.Config{})
    if err != nil {
		log.Fatal(err)
	}

    db.AutoMigrate(&User{})
    return db
}
