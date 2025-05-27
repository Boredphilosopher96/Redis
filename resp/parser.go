package resp

import (
	"bytes"
	"fmt"
	"strconv"
)

func parseCommand(msg []byte) {
}

// in the initial version, I am assuming all commands just sent as array args
// this is not true of the RESP	protocol, but REDIS clients do use this format
// So this serves as a good starting point
func Parse(msg []byte) []byte {
	isPing := bytes.HasPrefix(msg, []byte("*1\r\n$4\r\nping\r\n"))
	isEcho := bytes.HasPrefix(msg, []byte("*2\r\n$4\r\necho\r\n"))
	fmt.Println("Parsing message: ", string(msg))
	if isPing {
		return SendSimpleString("PONG")
	} else if isEcho {
		return SendSimpleString(string(msg[18:]))
	}
	return SendSimpleString("SimplySafe")
}

func SendError(msg string) []byte {
	return []byte("-ERR " + msg + "\r\n")
}

func SendBlobString(msg string) []byte {
	return []byte("$" + fmt.Sprintf("%d", len(msg)) + "\r\n" + msg + "\r\n")
}

func SendSimpleString(msg string) []byte {
	return []byte("+" + msg + "\r\n")
}

func SendInteger(msg int64) []byte {
	return []byte(":" + strconv.FormatInt(msg, 10) + "\r\n")
}

func SendNull() []byte {
	return []byte("_\r\n")
}

func SendDouble(msg float64) []byte {
	return []byte("," + strconv.FormatFloat(msg, 'f', -1, 64) + "\r\n")
}

type Token int

const (
	ping Token = iota
	echo
)

type Command struct {
	token Token
	value any
}

func parseArray(msg []byte) ([]byte, []byte) {
	if !bytes.HasPrefix(msg, []byte{'*'}) {
		return nil, SendError("Not a valid array")
	}

	return nil, nil
}

func parseMap(msg []byte) {
}

func parseString(msg []byte) (string, []byte) {
	if !bytes.HasPrefix(msg, []byte{'+'}) {
		return "", SendError("Not a valid string")
	}
	return "", nil
}

func parseBulkString() {}

func parseInteger(msg []byte) (int64, []byte) {
	if !bytes.HasPrefix(msg, []byte{':'}) {
		return -1, SendError("Not a valid integer")
	}
	return 1, nil
}

func parseDouble(msg []byte) (float64, []byte) {
	if !bytes.HasPrefix(msg, []byte{','}) {
		return -1.0, SendError("Not a valid floating point number")
	}
	return 0.123, nil
}

func parseNull(msg []byte) ([]byte, []byte) {
	if !bytes.HasPrefix(msg, []byte{'_'}) {
		return nil, SendError("Not a valid null")
	}
	return SendNull(), nil
}

func Serialize() {
}

func Deserialize() (c *Command) {
	return &Command{}
}
