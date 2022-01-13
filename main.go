package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "gorm.io/gorm"
)

const (
    WORKERS_COUNT = 2
)

func worker(id int, assignClnt <-chan Client, workerMsg chan<- string, db *gorm.DB) {
    log.Printf("Worker %d starting\n", id)
    for {
        workerMsg <- "ready"
        clnt := <- assignClnt
        log.Printf("Worker %d got job\n", id)
        handleClient(id, clnt, db)
        log.Printf("Worker %d finished job\n", id)
    }
}

func main() {
    db := startDB()

    listener, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Args[1]))
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    assignClnt := make(chan Client)
    workerMsg := make(chan string)

    for i := 0; i < WORKERS_COUNT; i++ {
        i := i
        go func() {
            worker(i, assignClnt, workerMsg, db)
        }()
    }

    for {
        <- workerMsg
        for {
            conn, err := listener.Accept()
            if err != nil {
                log.Println(err)
                continue
            }
            assignClnt <- Client{conn: conn}
            break
        }
    }
}
