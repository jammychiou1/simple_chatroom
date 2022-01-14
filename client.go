package main

import (
    "net"
    "bufio"
    "strings"
    "fmt"
    "gorm.io/gorm"
    "errors"
    "log"
    "encoding/base64"
    "github.com/mattn/go-sqlite3"
    "strconv"
)

type Client struct {
    conn net.Conn
    r *bufio.Reader
    w *bufio.Writer

    username string
    userID int
    chatroomID int
}

type handleFunc func (id int, clnt *Client, db *gorm.DB) handleFunc

func handleInit(id int, clnt *Client, db *gorm.DB) (next handleFunc) {
    for {
        ln, err := clnt.r.ReadString('\n')
        if err != nil {
            log.Printf("Worker %d got EOF\n", id)
            return nil
        }
        ln = strings.TrimSpace(ln)
        tokens := strings.Split(ln, " ")
        if tokens[0] == "login" && len(tokens) == 3 {
            log.Printf("Client at Worker %d wants to login\n", id)
            usernameBytes, err1 := base64.StdEncoding.DecodeString(tokens[1])
            passwordBytes, err2 := base64.StdEncoding.DecodeString(tokens[2])
            if err1 != nil || err2 != nil {
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            username := string(usernameBytes)
            password := string(passwordBytes)
            
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
            log.Printf("Client at Worker %d registered as user: %v\n", id, user)
            fmt.Fprintf(clnt.w, "yes\n")
            clnt.w.Flush()
            clnt.username = username
            clnt.userID = user.ID
            break
        } else if tokens[0] == "register" && len(tokens) == 3 {
            log.Printf("Client at Worker %d wants to register\n", id)
            usernameBytes, err1 := base64.StdEncoding.DecodeString(tokens[1])
            passwordBytes, err2 := base64.StdEncoding.DecodeString(tokens[2])
            if err1 != nil || err2 != nil {
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            username := string(usernameBytes)
            password := string(passwordBytes)

            user := User{Username: username, Password: password};
            if err := db.Create(&user).Error; err != nil {
                //log.Printf("%T\n", err)
                var sqliteErr sqlite3.Error
                if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
                    log.Printf("Client at Worker %d register duplicate username\n", id)
                } else {
                    log.Printf("Worker %d got unkown db error\n", id)
                }
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            log.Printf("Client at Worker %d registered as user: %v\n", id, user)
            fmt.Fprintf(clnt.w, "yes\n")
            clnt.w.Flush()
            clnt.username = username
            clnt.userID = user.ID
            break

        }
        log.Printf("Client at Worker %d sent unknown command\n", id)
        fmt.Fprintf(clnt.w, "unknown command\n")
        clnt.w.Flush()
        return nil
    }
    return handleCommand
}

func handleCommand(id int, clnt *Client, db *gorm.DB) (next handleFunc) {
    for {
        ln, err := clnt.r.ReadString('\n')
        if err != nil {
            log.Printf("Worker %d got EOF\n", id)
            return nil
        }
        ln = strings.TrimSpace(ln)
        tokens := strings.Split(ln, " ")
        log.Println(tokens)
        if tokens[0] == "listFriends" && len(tokens) == 1 {
            user := User{ID: clnt.userID}
            var friends []*User
            db.Model(&user).Association("Friends").Find(&friends)
            friendNames := make([]string, len(friends))
            for i, friend := range friends {
                friendNames[i] = base64.StdEncoding.EncodeToString([]byte(friend.Username))
            }
            log.Printf("Client at Worker %d list friends: %v\n", id, friendNames)
            fmt.Fprintln(clnt.w, strings.Join(friendNames, " "))
            clnt.w.Flush()
            continue
        } else if tokens[0] == "addFriend" && len(tokens) == 2 {
            friendNameBytes, err := base64.StdEncoding.DecodeString(tokens[1])
            if err != nil {
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            friendName := string(friendNameBytes)

            user := User{ID: clnt.userID}
            err = db.Transaction(func(tx *gorm.DB) error {
                var friend User
                if err := db.Where("username = ?", friendName).Take(&friend).Error; err != nil {
                    if errors.Is(err, gorm.ErrRecordNotFound) {
                        return fmt.Errorf("nonexistFriend: %w", err)
                    }
                    return fmt.Errorf("unknown: %w", err)
                }
                
                var friends []*User
                if err := db.Model(&user).Association("Friends").Find(&friends, "friend_id = ?", friend.ID); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                if len(friends) >= 1 {
                    return fmt.Errorf("addedFriend")
                }

                if err := db.Model(&user).Association("Friends").Append(&friend); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                return nil
            })
            if err != nil {
                if strings.HasPrefix(err.Error(), "nonexistFriend") {
                    fmt.Fprintf(clnt.w, "nonexist\n")
                    clnt.w.Flush()
                } else if strings.HasPrefix(err.Error(), "addedFriend") {
                    fmt.Fprintf(clnt.w, "added\n")
                    clnt.w.Flush()
                } else {
                    fmt.Fprintf(clnt.w, "nonexist\n")
                    clnt.w.Flush()
                    log.Println(err)
                }
                continue
            }
            fmt.Fprintf(clnt.w, "ok\n")
            clnt.w.Flush()
            continue
        } else if tokens[0] == "deleteFriend" && len(tokens) == 2 {
            friendNameBytes, err := base64.StdEncoding.DecodeString(tokens[1])
            if err != nil {
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            friendName := string(friendNameBytes)

            user := User{ID: clnt.userID}
            err = db.Transaction(func(tx *gorm.DB) error {
                var friend User
                if err := db.Where("username = ?", friendName).Take(&friend).Error; err != nil {
                    if errors.Is(err, gorm.ErrRecordNotFound) {
                        return fmt.Errorf("nonexistFriend: %w", err)
                    }
                    return fmt.Errorf("unknown: %w", err)
                }
                
                //var friends []*User
                //if err := db.Model(&user).Association("Friends").Find(&friends, "friend_id = ?", friend.ID); err != nil {
                //    return fmt.Errorf("unknown: %w", err)
                //}
                //if len(friends) >= 1 {
                //    return fmt.Errorf("addedFriend")
                //}

                if err := db.Model(&user).Association("Friends").Delete(&friend); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                return nil
            })
            if err != nil {
                fmt.Fprintf(clnt.w, "failed\n")
                clnt.w.Flush()
                log.Println(err)
                continue
            }
            fmt.Fprintf(clnt.w, "ok\n")
            clnt.w.Flush()
            continue
        } else if tokens[0] == "createChatroom" && len(tokens) == 2 {
            friendNameBytes, err := base64.StdEncoding.DecodeString(tokens[1])
            if err != nil {
                fmt.Fprintf(clnt.w, "no\n")
                clnt.w.Flush()
                continue
            }
            friendName := string(friendNameBytes)

            user := User{ID: clnt.userID}
            chatroom := Chatroom{Name: "chatroom"}
            err = db.Transaction(func(tx *gorm.DB) error {
                var friend User
                if err := db.Where("username = ?", friendName).Take(&friend).Error; err != nil {
                    if errors.Is(err, gorm.ErrRecordNotFound) {
                        return fmt.Errorf("nonexistFriend: %w", err)
                    }
                    return fmt.Errorf("unknown: %w", err)
                }
                
                var friends []*User
                if err := db.Model(&user).Association("Friends").Find(&friends, "friend_id = ?", friend.ID); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                if len(friends) == 0 {
                    return fmt.Errorf("notFriend")
                }

                if err := db.Model(&user).Association("Chatrooms").Append(&chatroom); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                if err := db.Model(&friend).Association("Chatrooms").Append(&chatroom); err != nil {
                    return fmt.Errorf("unknown: %w", err)
                }
                return nil
            })
            if err != nil {
                fmt.Fprintf(clnt.w, "failed\n")
                clnt.w.Flush()
                log.Println(err)
                continue
            }
            fmt.Fprintf(clnt.w, "ok %d\n", chatroom.ID)
            clnt.w.Flush()
            continue
        } else if tokens[0] == "listChatroom" && len(tokens) == 1 {
            user := User{ID: clnt.userID}
            var chatrooms []*Chatroom
            if err := db.Preload("Users").Model(&user).Association("Chatrooms").Find(&chatrooms); err != nil {
                fmt.Fprintln(clnt.w, "")
                continue
            }
            fmt.Fprintln(clnt.w, len(chatrooms))
            log.Printf("Client at Worker %d listChatroom: %v\n", id, len(chatrooms))
            for _, chatroom := range chatrooms {
                chatroomUsers := make([]string, len(chatroom.Users))
                for j, chatroomUser := range chatroom.Users {
                    chatroomUsers[j] = base64.StdEncoding.EncodeToString([]byte(chatroomUser.Username))
                }
                fmt.Fprintf(clnt.w, "%d %s\n", chatroom.ID, strings.Join(chatroomUsers, " "))
                log.Printf("Client at Worker %d listChatroom: %d %s\n", id, chatroom.ID, strings.Join(chatroomUsers, " "))
            }
            clnt.w.Flush()
            continue
        } else if tokens[0] == "joinChatroom" && len(tokens) == 2 {
            chatroomID, err := strconv.Atoi(tokens[1])
            user := User{ID: clnt.userID}
            if err != nil {
                fmt.Fprintln(clnt.w, "failed")
                clnt.w.Flush()
                log.Println(err)
                continue
            }
            var chatrooms []*Chatroom
            if err := db.Preload("Users").Model(&user).Association("Chatrooms").Find(&chatrooms, "chatroom_id = ?", chatroomID); err != nil {
                fmt.Fprintln(clnt.w, "failed")
                clnt.w.Flush()
                log.Println(err)
                continue
            }
            if len(chatrooms) != 1 {
                fmt.Fprintln(clnt.w, "failed")
                clnt.w.Flush()
                continue
            }
            chatroom := chatrooms[0]
            chatroomUsers := make([]string, len(chatroom.Users))
            for j, chatroomUser := range chatroom.Users {
                chatroomUsers[j] = base64.StdEncoding.EncodeToString([]byte(chatroomUser.Username))
            }
            fmt.Fprintf(clnt.w, "ok %s\n", strings.Join(chatroomUsers, " "))
            log.Printf("Client at Worker %d joinChatroom: %d %s\n", id, chatroom.ID, strings.Join(chatroomUsers, " "))
            clnt.w.Flush()
            clnt.chatroomID = chatroom.ID
            continue
        } else if tokens[0] == "send" && len(tokens) == 2 {
            if clnt.chatroomID == -1 {
                continue
            }
            textEnc := tokens[1]
            chatroom := Chatroom{ID: clnt.chatroomID}
            text, err := base64.StdEncoding.DecodeString(textEnc)
            if err != nil {
                continue
            }
            log.Printf("Client at Worker %d sent: %q\n", id, text)
            message := Message{Content: fmt.Sprintf("text %s", textEnc)}
            if err := db.Model(&chatroom).Association("Messages").Append(&message); err != nil {
                log.Println(err)
                continue
            }
            continue
        } else if tokens[0] == "logs" && len(tokens) == 3 {
            if clnt.chatroomID == -1 {
                continue
            }
            begin, err1 := strconv.Atoi(tokens[1])
            end, err2 := strconv.Atoi(tokens[2])
            if err1 != nil || err2 != nil {
                continue
            }
            var limit int
            if end == -1 {
                limit = -1
            } else {
                limit = end - begin
            }
            offset := begin
            var messages []Message
            if err := db.Model(&Message{}).Offset(offset).Limit(limit).Find(&messages, "chatroom_id = ?", clnt.chatroomID).Error; err != nil {
                log.Println(err)
                continue
            }
            newEnd := begin + len(messages)
            log.Println(messages)
            fmt.Fprintf(clnt.w, "%d %d\n", begin, newEnd)
            log.Printf("Client at %d logs: %d %d\n", id, begin, newEnd)
            for _, message := range messages {
                fmt.Fprintf(clnt.w, "%s\n", message.Content)
                log.Printf("Cleint at %d logs: %s\n", id, message.Content)
            }
            clnt.w.Flush()
            continue
            
        }
        log.Printf("Client at Worker %d sent unknown command\n", id)
        fmt.Fprintf(clnt.w, "unknown command\n")
        clnt.w.Flush()
        return nil
    }
}

func handleClient(id int, clnt Client, db *gorm.DB) {
    defer clnt.conn.Close()
    clnt.r = bufio.NewReader(clnt.conn)
    clnt.w = bufio.NewWriter(clnt.conn)
    clnt.chatroomID = -1
    next := handleInit(id, &clnt, db)
    if next == nil {
        return
    }
    next(id, &clnt, db)
}

