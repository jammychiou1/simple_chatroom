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
    Chatrooms []*Chatroom `gorm:"many2many:user_chatrooms"`
}

type Chatroom struct {
    ID int
    Users []*User `gorm:"many2many:user_chatrooms"`
    Messages []Message
    Files []File
    Name string `gorm:"many2many:user_chatrooms"`
}

type Message struct {
    ID int
    Content string `gorm:"not null"`
    ChatroomID int
}

type File struct {
    ID int
    Handle string `gorm:"not null;unique"`
    Filename string `gorm:"not null"`
    ChatroomID int
    UserID int
    Uploaded bool `gorm:"default:false"`
    Filesize int
    Type int
}

const (
    DB_FILENAME = "mydb.db"
)

const (
    TYPE_FILE = 1
    TYPE_IMAGE = 2
)

func typeName(tp int) string {
    if tp == 1 {
        return "file"
    } else {
        return "image"
    }
}

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

    db.AutoMigrate(&User{}, &Chatroom{}, &Message{}, &File{})
    return db
}
