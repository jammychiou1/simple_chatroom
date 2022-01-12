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

func worker(id int, assignJob <-chan Client, workerMsg chan<- string, db *gorm.DB) {
    log.Printf("Worker %d starting\n", id)
    for {
        workerMsg <- "ready"
        job := <- assignJob
        log.Printf("Worker %d got job\n", id)
        handleClient(id, job, db)
    }
}

func main() {
    db := startDB()

    listener, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Args[1]))
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    assignJob := make(chan Client)
    workerMsg := make(chan string)

    for i := 0; i < WORKERS_COUNT; i++ {
        i := i
        go func() {
            worker(i, assignJob, workerMsg, db)
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
            assignJob <- Client{conn: conn}
            break
        }
    }
}
