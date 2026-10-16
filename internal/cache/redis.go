package cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

// Redis is a minimal Redis client (RESP2 protocol, GET/SET/INCR) with a small connection pool.
// Errors are treated as cache misses: the cache must never break the request path.
type Redis struct {
	addr    string
	timeout time.Duration
	pool    chan *conn
	// OnError is called for every failed command (e.g. to count errors in metrics).
	OnError func(error)
}

type conn struct {
	net.Conn
	r *bufio.Reader
}

// NewRedis creates a client for addr (host:port) with up to size pooled connections.
func NewRedis(addr string, size int) *Redis {
	return &Redis{addr: addr, timeout: 200 * time.Millisecond, pool: make(chan *conn, size), OnError: func(error) {}}
}

var errNil = errors.New("redis: nil")

func (c *Redis) get() (*conn, error) {
	select {
	case cn := <-c.pool:
		return cn, nil
	default:
		nc, err := net.DialTimeout("tcp", c.addr, c.timeout)
		if err != nil {
			return nil, err
		}
		return &conn{Conn: nc, r: bufio.NewReader(nc)}, nil
	}
}

func (c *Redis) put(cn *conn) {
	select {
	case c.pool <- cn:
	default:
		_ = cn.Close()
	}
}

// do sends one command and returns the reply (bulk string, integer or status as bytes).
func (c *Redis) do(args ...string) ([]byte, error) {
	cn, err := c.get()
	if err != nil {
		return nil, err
	}
	_ = cn.SetDeadline(time.Now().Add(c.timeout))
	reply, err := roundTrip(cn, args)
	if err != nil && !errors.Is(err, errNil) {
		_ = cn.Close() // the connection state is unknown
		return nil, err
	}
	c.put(cn)
	return reply, err
}

func roundTrip(cn *conn, args []string) ([]byte, error) {
	buf := []byte("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, a := range args {
		buf = append(buf, "$"+strconv.Itoa(len(a))+"\r\n"+a+"\r\n"...)
	}
	if _, err := cn.Write(buf); err != nil {
		return nil, err
	}
	return readReply(cn.r)
}

func readReply(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 {
		return nil, fmt.Errorf("redis: short reply %q", line)
	}
	kind, body := line[0], line[1:len(line)-2]
	switch kind {
	case '+', ':':
		return []byte(body), nil
	case '-':
		return nil, errors.New("redis: " + body)
	case '$':
		return readBulk(r, body)
	default:
		return nil, fmt.Errorf("redis: unexpected reply %q", line)
	}
}

func readBulk(r *bufio.Reader, size string) ([]byte, error) {
	n, err := strconv.Atoi(size)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, errNil
	}
	data := make([]byte, n+2)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (c *Redis) fail(err error) {
	if err != nil && !errors.Is(err, errNil) {
		c.OnError(err)
	}
}

// Get returns a cached value.
func (c *Redis) Get(_ context.Context, key string) ([]byte, bool) {
	v, err := c.do("GET", key)
	c.fail(err)
	return v, err == nil
}

// Set stores a value with a TTL.
func (c *Redis) Set(_ context.Context, key string, val []byte, ttl time.Duration) {
	_, err := c.do("SET", key, string(val), "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	c.fail(err)
}

// Version returns the current version of a scope (0 if unknown or on errors).
func (c *Redis) Version(_ context.Context, scope string) int64 {
	v, err := c.do("GET", "ver:"+scope)
	c.fail(err)
	n, _ := strconv.ParseInt(string(v), 10, 64)
	return n
}

// Bump invalidates every entry of a scope.
func (c *Redis) Bump(_ context.Context, scope string) {
	_, err := c.do("INCR", "ver:"+scope)
	c.fail(err)
}
