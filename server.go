package main

import (
    "net"
    "fmt"
    "time"
    "math/rand"
    "encoding/json"
)

type Command struct {
    DesiredState int
    Enforced bool
    StartEnforce int64
    EndEnforce int64
}

func printMessages(msgchan <-chan string) {
    for {
        msg := <-msgchan
        fmt.Printf("JSON received: %s\n", msg)
    }
}

// Return Command formatted as a JSON string
func createJSON(currentCommand Command) string {
    m, _ := json.Marshal(currentCommand)
    return string(m)
}

// Return the current time in epoch format
func epochTime() int64 {
    now := time.Now()
    return now.Unix()
}

func sendCommand(conn net.Conn) {
    for {
        desiredState := rand.Intn(2)
        var currentCommand = Command{desiredState, false, epochTime(), epochTime()+600}
        fmt.Fprintf(conn, createJSON(currentCommand))
        time.Sleep(5 * time.Second)
    }
}

func handleConnection(c net.Conn, msgchan chan<- string) {
    buf := make([]byte, 4096)
    go sendCommand(c)

    for {
        // Read from the socket connection and send to the channel
        n, err := c.Read(buf)
        if err != nil || n == 0 {
            c.Close()
            break
        }
        msgchan <- string(buf[0:n])

        // Send back a random message
        //fmt.Fprintf(c,"Sending random message")

    }
    fmt.Printf("Connection from %v closed.\n", c.RemoteAddr())
}

func main() {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        // handle error
    }
    
    msgchan := make(chan string)
    go printMessages(msgchan)
    
    for {
        conn, err := ln.Accept()
        if err != nil {
            // handle error
        }
        go handleConnection(conn, msgchan)
    }
}
