package main

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/tbenz9/cec"
    "time"
    "fmt"
    "os"
    "math/rand"
    "net"
    "strings"
    "encoding/json"
)

/////////////////////////////////////////////////////////////
//
// Database Functions and Variables
//
/////////////////////////////////////////////////////////////

var localDatabasePath string = "/tmp/eyes/eyesd.db"

func insertStateIntoDatabase(state int, timeChanged int64) int64 {
    db, err := sql.Open("sqlite3", localDatabasePath)
 
    stmt, err := db.Prepare("INSERT INTO state (state, timestamp) values(?,?)")
    res, err := stmt.Exec(state, timeChanged)

    id, err := res.LastInsertId()
    if debug {fmt.Printf("Database returned ID %v\n", id)}
    db.Close()
    checkErr(err)
    return id
}

func setupLocalDatabase() {
    err := os.MkdirAll("/tmp/eyes",0755)
    db, err := sql.Open("sqlite3", localDatabasePath)

    _, err = db.Exec("CREATE TABLE IF NOT EXISTS state (metricsID INTEGER PRIMARY KEY, timestamp INTEGER NOT NULL, state INTEGER NOT NULL)")
    checkErr(err)
    db.Close()
}

/////////////////////////////////////////////////////////////
//
// Websocket Functions
//
/////////////////////////////////////////////////////////////

var remoteServerAddress string = "192.168.0.106"

func setupWebsocket(outgoing chan string, incoming chan string) {
    conn, err := net.Dial("tcp", "192.168.0.106:8080")
    go sendToServer(conn, outgoing)
    go receiveFromServer(conn, incoming)
    checkErr(err)
}

// Send whatever is on the outgoing channel to the server
func sendToServer(conn net.Conn, outgoing chan string) {
    for { fmt.Fprintf(conn, <-outgoing) }
}

func receiveFromServer(conn net.Conn, incoming chan string) {
    buf := make([]byte, 4096)
    for {
        msg, err :=conn.Read(buf)
        checkErr(err)
        go executeCommand(decodeJSON(buf[0:msg]))
    }
}

/////////////////////////////////////////////////////////////
//
// Command Functions and Variables
//
/////////////////////////////////////////////////////////////

func executeCommand(currentCommand Command) {
    if currentCommand.Enforced {
        // Add to enforcement table
    } else if !currentCommand.Enforced {
        // Immediately execute command
        if currentCommand.DesiredState == 0 {
            if emulate {state = 0}
            if !emulate {cec.PowerOn(0)}
        } else if currentCommand.DesiredState == 1 {
            if emulate {state = 1}
            if !emulate {cec.Standby(0)}
        }
    } else { fmt.Println("Bad data from server") }

}

/////////////////////////////////////////////////////////////
//
// Other Functions and Variables
//
/////////////////////////////////////////////////////////////

var debug bool = false
var emulate bool = true
var sleepTime int = 2
var state int = 0

// Do I even need this struct?
type Device struct {
    Identifier string
    CurrentState int
    DatabaseID int64
    CurrentTime int64
}

type Command struct {
    DesiredState int
    Enforced bool
    StartEnforce int64
    EndEnforce int64
}

func changedState(state int, outgoing chan string, timestamp int64) {

    id := insertStateIntoDatabase(state, timestamp)
    var currentDevice = Device{macAddress(), state, id, timestamp}

    outgoing <- (createJSON(currentDevice))
    if debug {fmt.Printf("TV Changed State to %v\n", state)}
}

// Return the current time in epoch format
func epochTime() int64 {
    now := time.Now()
    return now.Unix()
}

// Return the MAC address of eth0
func macAddress() string {
    mac, err := net.InterfaceByName("eth0")
    checkErr(err)
    return strings.ToUpper(mac.HardwareAddr.String())
}

func decodeJSON(b []byte) Command {
        var m Command
        err := json.Unmarshal(b, &m)
        if debug {fmt.Printf("received %v from server.\n",m)}
        checkErr(err)
        return m
}

// Return Device formatted as a JSON string
func createJSON(currentDevice Device) string {
    m, err := json.Marshal(currentDevice)
    if debug {fmt.Printf("JSON is: %v\n", string(m))}
    checkErr(err)
    return string(m)
}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

/////////////////////////////////////////////////////////////
//
// Main
//
/////////////////////////////////////////////////////////////

func main() {

    if !emulate {cec.Open("", "cec.go")}
    previousState := 5
    outgoing := make(chan string)
    incoming := make(chan string)

    // Initial Startup tasks
    go setupWebsocket(outgoing, incoming)
    go setupLocalDatabase()

    // Check the TV state every second
    for {
        if !emulate {state = cec.GetDevicePowerStatus(0)}
        if emulate {state = rand.Intn(2)}
        if debug {fmt.Printf("The TV is %v\n", state)}
        if state != previousState {
            go changedState(state, outgoing, epochTime())
        }
        previousState = state
        time.Sleep(1 * time.Second)
        if emulate {time.Sleep(time.Duration(sleepTime) * time.Second)}
    }
}
