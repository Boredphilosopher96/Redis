package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strings"
)

func main() {
    conn, err := net.Dial("tcp",":6379")
    defer conn.Close()
    if err != nil {
        fmt.Println("Could not create connection ", err)
        return
    }
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if err != nil {
            fmt.Println("error reading input ", err)
        }
        _, err := conn.Write([]byte(line))
        if err != nil {
            fmt.Println("Error writing to server ",err)
            return
        }
        buff := make([]byte, 1024)
        _, error := conn.Read(buff)
        if error != nil {
            fmt.Println("Error reading from server ", err)
            os.Exit(1)
        }
        fmt.Println("Received ", string(buff))
    } 
}
