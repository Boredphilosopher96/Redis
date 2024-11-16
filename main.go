package main

import (
	"fmt"
	"net"
)


func main() {
    ln, err := net.Listen("tcp", ":6379")
    fmt.Println("Listening on 6379")
    if err != nil {
        fmt.Println("there is some error here ", err)
        return
    }
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println("Could not create connection ",err)
            return
        }
        go handleConn(conn)
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    fmt.Println("Handling connection")
    for {
        buff := make([]byte, 1024)
        _ , err := conn.Read(buff)
        if err != nil {
            fmt.Println("Could not read message ", err)
            return
        }
        fmt.Printf("Received: %s \n", buff)
    }
}
