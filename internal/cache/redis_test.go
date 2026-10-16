package cache

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRedis is an in-test RESP server supporting GET, SET (PX ignored), INCR and FAIL.
func fakeRedis(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var mu sync.Mutex
	data := map[string]string{}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go serve(c, &mu, data)
		}
	}()
	return ln.Addr().String()
}

func serve(c net.Conn, mu *sync.Mutex, data map[string]string) {
	defer func() { _ = c.Close() }()
	r := bufio.NewReader(c)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}
		mu.Lock()
		reply := execute(args, data)
		mu.Unlock()
		if _, err := io.WriteString(c, reply); err != nil {
			return
		}
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
	if n == 0 {
		return nil, io.EOF
	}
	args := make([]string, n)
	for i := range args {
		size, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		l, _ := strconv.Atoi(strings.TrimSpace(size[1:]))
		buf := make([]byte, l+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args[i] = string(buf[:l])
	}
	return args, nil
}

func execute(args []string, data map[string]string) string {
	switch args[0] {
	case "GET":
		v, ok := data[args[1]]
		if !ok {
			return "$-1\r\n"
		}
		return "$" + strconv.Itoa(len(v)) + "\r\n" + v + "\r\n"
	case "SET":
		data[args[1]] = args[2]
		return "+OK\r\n"
	case "INCR":
		n, _ := strconv.Atoi(data[args[1]])
		data[args[1]] = strconv.Itoa(n + 1)
		return ":" + data[args[1]] + "\r\n"
	case "GARBAGE":
		return "?\r\n"
	default:
		return "-ERR unknown command\r\n"
	}
}

func TestRedisCacheAside(t *testing.T) {
	c := NewRedis(fakeRedis(t), 2)
	ctx := context.Background()
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("empty cache must miss")
	}
	c.Set(ctx, "k", []byte("v\r\nwith crlf"), time.Minute)
	if v, ok := c.Get(ctx, "k"); !ok || string(v) != "v\r\nwith crlf" {
		t.Fatalf("got %q %v", v, ok)
	}
	if c.Version(ctx, "t1") != 0 {
		t.Fatal("unknown scope version must be 0")
	}
	c.Bump(ctx, "t1")
	c.Bump(ctx, "t1")
	if c.Version(ctx, "t1") != 2 {
		t.Fatal("bump must increment the scope version")
	}
}

func TestRedisErrorsAreMisses(t *testing.T) {
	var errs []error
	c := NewRedis(fakeRedis(t), 1)
	c.OnError = func(err error) { errs = append(errs, err) }
	if _, err := c.do("NOPE"); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("error reply: %v", err)
	}
	if _, err := c.do("GARBAGE"); err == nil {
		t.Fatal("unexpected reply type must fail")
	}
	down := NewRedis("127.0.0.1:1", 1)
	down.OnError = c.OnError
	if _, ok := down.Get(context.Background(), "k"); ok {
		t.Fatal("unreachable redis must miss")
	}
	down.Set(context.Background(), "k", nil, time.Second)
	if len(errs) != 2 {
		t.Fatalf("errors reported: %v", errs)
	}
	var n Nop
	n.Set(context.Background(), "k", nil, 0)
	n.Bump(context.Background(), "s")
	if _, ok := n.Get(context.Background(), "k"); ok || n.Version(context.Background(), "s") != 0 {
		t.Fatal("nop cache")
	}
	if _, err := readReply(bufio.NewReader(strings.NewReader("$x\r\n"))); err == nil {
		t.Fatal("bad bulk length")
	}
	if _, err := readReply(bufio.NewReader(strings.NewReader("$5\r\nab"))); err == nil {
		t.Fatal("truncated bulk")
	}
	if _, err := readReply(bufio.NewReader(strings.NewReader("+\n"))); err == nil {
		t.Fatal("short reply")
	}
	if !errors.Is(func() error { _, e := readBulk(nil, "-1"); return e }(), errNil) {
		t.Fatal("nil bulk")
	}
}

func TestRedisPoolOverflowClosesConnections(t *testing.T) {
	c := NewRedis(fakeRedis(t), 1)
	a, _ := c.get()
	b, _ := c.get()
	c.put(a)
	c.put(b) // pool is full: b is closed
	if _, err := b.Write([]byte("PING")); err == nil {
		t.Fatal("overflow connection must be closed")
	}
}
