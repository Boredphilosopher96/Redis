package resp

import (
	"bytes"
	"testing"
)

// Helper builders
func arr(parts ...string) []byte {
	// Build RESP array of bulk strings
	buf := []byte("*")
	buf = append(buf, []byte{byte('0' + (len(parts) / 10)), byte('0' + (len(parts) % 10))}...)
	// remove leading zero when < 10
	if len(parts) < 10 {
		buf = buf[:1]
		buf = append(buf, []byte{byte('0' + len(parts))}...)
	}
	buf = append(buf, []byte("\r\n")...)
	for _, p := range parts {
		buf = append(buf, []byte("$")...)
		buf = append(buf, []byte(stringInt(len(p)))...)
		buf = append(buf, []byte("\r\n")...)
		buf = append(buf, []byte(p)...)
		buf = append(buf, []byte("\r\n")...)
	}
	return buf
}

func stringInt(n int) string {
	if n == 0 {
		return "0"
	}
	s := []byte{}
	for n > 0 {
		d := n % 10
		s = append([]byte{byte('0' + d)}, s...)
		n /= 10
	}
	return string(s)
}

func TestRESP3PingNoArg(t *testing.T) {
	cmd := arr("PING")
	got := Parse(cmd)
	exp := []byte("+PONG\r\n")
	if !bytes.Equal(got, exp) {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}

func TestRESP3PingWithArg(t *testing.T) {
	cmd := arr("PING", "hello")
	got := Parse(cmd)
	exp := []byte("$5\r\nhello\r\n")
	if !bytes.Equal(got, exp) {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}

func TestRESP3Echo(t *testing.T) {
	cmd := arr("ECHO", "world")
	got := Parse(cmd)
	exp := []byte("$5\r\nworld\r\n")
	if !bytes.Equal(got, exp) {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}

func TestRESP3SetGet(t *testing.T) {
	set := arr("SET", "k", "v")
	got := Parse(set)
	expOK := []byte("+OK\r\n")
	if !bytes.Equal(got, expOK) {
		t.Fatalf("expected %q, got %q", expOK, got)
	}
	get := arr("GET", "k")
	got = Parse(get)
	exp := []byte("$1\r\nv\r\n")
	if !bytes.Equal(got, exp) {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}

func TestRESP3GetMissingReturnsNull(t *testing.T) {
	get := arr("GET", "missing")
	got := Parse(get)
	exp := []byte("_\r\n")
	if !bytes.Equal(got, exp) {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}

func TestHelloDefault(t *testing.T) {
	cmd := arr("HELLO")
	got := Parse(cmd)
	// Expect a RESP3 map starting with % and containing fields; we assert prefixes
	if len(got) == 0 || got[0] != '%' {
		t.Fatalf("expected map reply, got %q", got)
	}
	if !bytes.Contains(got, []byte("+server\r\n+redis\r\n")) {
		t.Fatalf("expected server redis in HELLO map, got %q", got)
	}
	if !bytes.Contains(got, []byte("+proto\r\n:3\r\n")) {
		t.Fatalf("expected proto 3 in HELLO reply, got %q", got)
	}
}

func TestHelloProto3AuthSetName(t *testing.T) {
	cmd := arr("HELLO", "3", "AUTH", "default", "secret", "SETNAME", "client1")
	got := Parse(cmd)
	if len(got) == 0 || got[0] != '%' {
		t.Fatalf("expected map reply, got %q", got)
	}
	if !bytes.Contains(got, []byte("+proto\r\n:3\r\n")) {
		t.Fatalf("expected proto 3 in HELLO reply, got %q", got)
	}
}

func TestHelloUnsupportedProto(t *testing.T) {
	cmd := arr("HELLO", "2")
	got := Parse(cmd)
	expPrefix := []byte("-ERR ")
	if !bytes.HasPrefix(got, expPrefix) {
		t.Fatalf("expected error for unsupported proto, got %q", got)
	}
}
