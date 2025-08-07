package resp

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

//
// Minimal RESP2 request parser supporting arrays of bulk strings and inline commands.
//

// in the initial version, I am assuming all commands just sent as array args
// this is not true of the RESP	protocol, but REDIS clients do use this format
// So this serves as a good starting point
func Parse(msg []byte) []byte {
	// Trim trailing zero bytes commonly present in fixed-size network buffers
	msg = bytes.TrimRight(msg, "\x00")

	// First try RESP array of bulk strings (standard client request format)
	if len(msg) > 0 && msg[0] == '*' {
		elems, err := parseArrayOfBulkStrings(msg)
		if err != nil {
			return SendError("Parse error: " + err.Error())
		}
		return handleCommand(elems)
	}

	return SendError("Protocol error: unsupported request (expect RESP3 array)")
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

// RESP3 Array (used for values like modules in HELLO)
func SendArray(elements [][]byte) []byte {
	b := []byte("*" + strconv.Itoa(len(elements)) + "\r\n")
	for _, el := range elements {
		b = append(b, el...)
	}
	return b
}

// RESP3 Map builder from ordered entries
type MapEntry struct {
	Key   string
	Value []byte
}

func SendMap(entries []MapEntry) []byte {
	b := []byte("%" + strconv.Itoa(len(entries)) + "\r\n")
	for _, e := range entries {
		// Keys as simple strings
		b = append(b, SendSimpleString(e.Key)...)
		b = append(b, e.Value...)
	}
	return b
}

// parseArrayOfBulkStrings parses a RESP array whose elements are bulk strings
// and returns the decoded string elements.
func parseArrayOfBulkStrings(msg []byte) ([]string, error) {
	if !bytes.HasPrefix(msg, []byte{'*'}) {
		return nil, fmt.Errorf("not an array")
	}

	// Parse array length
	crlf := []byte("\r\n")
	idx := bytes.Index(msg, crlf)
	if idx == -1 {
		return nil, fmt.Errorf("malformed array: missing CRLF after length")
	}
	lengthStr := string(msg[1:idx])
	numElements, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", lengthStr)
	}

	elems := make([]string, 0, numElements)
	pos := idx + 2 // move past \r\n

	for i := 0; i < numElements; i++ {
		if pos >= len(msg) {
			return nil, fmt.Errorf("unexpected end of message while reading element %d", i)
		}
		if msg[pos] != '$' {
			return nil, fmt.Errorf("expected bulk string for element %d", i)
		}

		// parse bulk length
		headerStart := pos + 1
		headerEndRel := bytes.Index(msg[headerStart:], crlf)
		if headerEndRel == -1 {
			return nil, fmt.Errorf("malformed bulk string: missing CRLF after length (element %d)", i)
		}
		headerEnd := headerStart + headerEndRel
		bulkLenStr := string(msg[headerStart:headerEnd])
		bulkLen, err := strconv.Atoi(bulkLenStr)
		if err != nil {
			return nil, fmt.Errorf("invalid bulk string length '%s' (element %d)", bulkLenStr, i)
		}
		if bulkLen < 0 {
			return nil, fmt.Errorf("null bulk string not allowed in command array (element %d)", i)
		}

		// read data
		dataStart := headerEnd + 2 // skip \r\n
		dataEnd := dataStart + bulkLen
		if dataEnd+2 > len(msg) { // need space for trailing \r\n
			return nil, fmt.Errorf("bulk string overruns buffer (element %d)", i)
		}
		data := msg[dataStart:dataEnd]
		if !bytes.Equal(msg[dataEnd:dataEnd+2], crlf) {
			return nil, fmt.Errorf("bulk string missing trailing CRLF (element %d)", i)
		}
		elems = append(elems, string(data))
		pos = dataEnd + 2
	}

	return elems, nil
}

// handleCommand executes supported commands from parsed tokens
func handleCommand(tokens []string) []byte {
	if len(tokens) == 0 {
		return SendError("Protocol error: empty command")
	}
	cmd := strings.ToLower(tokens[0])

	switch cmd {
	case "hello":
		return handleHello(tokens)
	case "ping":
		// PING or PING <message>
		if len(tokens) == 1 {
			return SendSimpleString("PONG")
		}
		if len(tokens) == 2 {
			// In RESP2, PING with message returns a bulk string echoing the message
			return SendBlobString(tokens[1])
		}
		return SendError("wrong number of arguments for 'ping' command")

	case "echo":
		if len(tokens) != 2 {
			return SendError("wrong number of arguments for 'echo' command")
		}
		return SendBlobString(tokens[1])

	case "set":
		if len(tokens) != 3 {
			return SendError("wrong number of arguments for 'set' command")
		}
		kvStore.Store(tokens[1], tokens[2])
		return SendSimpleString("OK")

	case "get":
		if len(tokens) != 2 {
			return SendError("wrong number of arguments for 'get' command")
		}
		if v, ok := kvStore.Load(tokens[1]); ok {
			return SendBlobString(fmt.Sprintf("%v", v))
		}
		return SendNull()
	default:
		return SendError(fmt.Sprintf("unknown command '%s'", cmd))
	}
}

// simple concurrency-safe in-memory store for SET/GET
var kvStore sync.Map

// HELLO handling (RESP3 handshake)
var nextClientID uint64

func handleHello(tokens []string) []byte {
	// Accept: HELLO, or HELLO 3 [AUTH user pass] [SETNAME name]
	proto := 3
	i := 1
	if len(tokens) >= 2 {
		// protover provided
		n, err := strconv.Atoi(tokens[1])
		if err != nil {
			return SendError("syntax error")
		}
		if n != 3 {
			return SendError("unsupported protover")
		}
		proto = n
		i = 2
	}

	for i < len(tokens) {
		arg := strings.ToLower(tokens[i])
		switch arg {
		case "auth":
			if i+2 >= len(tokens) {
				return SendError("wrong number of arguments for 'hello' command")
			}
			// username := tokens[i+1]; password := tokens[i+2]
			// No-op authentication in this toy server
			i += 3
		case "setname":
			if i+1 >= len(tokens) {
				return SendError("wrong number of arguments for 'hello' command")
			}
			// name := tokens[i+1]
			i += 2
		default:
			return SendError("syntax error")
		}
	}

	id := atomic.AddUint64(&nextClientID, 1)

	// Build HELLO map reply
	entries := []MapEntry{
		{Key: "server", Value: SendSimpleString("redis")},
		{Key: "version", Value: SendSimpleString("0.0.1")},
		{Key: "proto", Value: SendInteger(int64(proto))},
		{Key: "id", Value: SendInteger(int64(id))},
		{Key: "mode", Value: SendSimpleString("standalone")},
		{Key: "role", Value: SendSimpleString("master")},
		{Key: "modules", Value: SendArray(nil)},
	}
	return SendMap(entries)
}
