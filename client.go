package main

import (
    "net"
    "bufio"
    "strings"
    "fmt"
    "gorm.io/gorm"
    "errors"
    "log"
)

type Client struct {
    conn net.Conn
    r *bufio.Reader
    w *bufio.Writer

    username string
}

func handleClient(id int, clnt Client, db *gorm.DB) {
    defer clnt.conn.Close()
    clnt.r = bufio.NewReader(clnt.conn)
    clnt.w = bufio.NewWriter(clnt.conn)
    for {
        ln, err := clnt.r.ReadString('\n')
        if err != nil {
            return
        }
        ln = strings.TrimSpace(ln)
        clnt.w.Flush()
        tokens := strings.Split(ln, " ")
        if tokens[0] == "login" && len(tokens) == 3 {
            log.Printf("Client at Worker %d wants to login\n", id)
            username := tokens[1]
            password := tokens[2]
            
            var user User
            if err := db.Where("username = ? AND password = ?", username, password).Take(&user).Error; err != nil {
                if errors.Is(err, gorm.ErrRecordNotFound) {
                    fmt.Fprintf(clnt.w, "no\n")
                    clnt.w.Flush()
                    continue
                }

                log.Println("db error", err)
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            fmt.Fprintf(clnt.w, "yes\n")
            clnt.w.Flush()
            clnt.username = username
            break
        } else if tokens[0] == "register" && len(tokens) == 3 {
            log.Printf("Client at Worker %d wants to register\n", id)
            username := tokens[1]
            password := tokens[2]

            var user User
            if err := db.Where("username = ?", username).Take(&user).Error; err != nil {
                if errors.Is(err, gorm.ErrRecordNotFound) {
                    err := db.Create(&User{Username: username, Password: password}).Error
                    if err == nil {
                        fmt.Fprintf(clnt.w, "yes\n")
                        clnt.w.Flush()
                        clnt.username = username
                        break
                    }
                }

                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                log.Println("db error", err)
                continue
            }
            fmt.Fprintf(clnt.w, "no\n")
            clnt.w.Flush()
            continue
        }
        fmt.Fprintf(clnt.w, "unknown command\n")
        clnt.w.Flush()
        return
    }
}

