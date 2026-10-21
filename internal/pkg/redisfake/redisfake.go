// Package redisfake is a tiny in-memory Redis server for tests. It speaks
// enough of the Redis protocol (RESP2) for the commands this service uses
// (GET, SET with EX/PX/NX, SETNX, INCR, DECR, EXPIRE, TTL, DEL, EXISTS, SCAN,
// PING), so repositories can be tested with the real go-redis client
// without a Redis server. Expiry is stored but never enforced. Do not use it
// outside tests.
package redisfake

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server is one in-memory Redis. Close it when the test ends.
type Server struct {
	ln    net.Listener
	mu    sync.Mutex
	data  map[string]string
	ttl   map[string]time.Duration
	cmds  []string
	conns map[net.Conn]struct{}
	wg    sync.WaitGroup
}

// Start listens on a free local port.
func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, data: map[string]string{}, ttl: map[string]time.Duration{}, conns: map[net.Conn]struct{}{}}
	s.wg.Add(1)
	go s.serve()
	return s, nil
}

// Addr is the host:port to give to the Redis client.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Close stops the server and drops open client connections.
func (s *Server) Close() {
	_ = s.ln.Close()
	s.mu.Lock()
	for c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// Get returns a stored value (for assertions).
func (s *Server) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	return v, ok
}

// TTL returns the expiry set on key (for assertions).
func (s *Server) TTL(key string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ttl[key]
}

// Set stores a value directly (test setup).
func (s *Server) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Keys returns every stored key, sorted.
func (s *Server) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Commands returns every command received, as "CMD arg1 arg2".
func (s *Server) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.cmds...)
}

func (s *Server) serve() {
	defer s.wg.Done()
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(c)
			s.mu.Lock()
			delete(s.conns, c)
			s.mu.Unlock()
		}()
	}
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}
		s.exec(w, args)
		if err := w.Flush(); err != nil {
			return
		}
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return strings.Fields(line), nil
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		hdr = strings.TrimRight(hdr, "\r\n")
		if !strings.HasPrefix(hdr, "$") {
			return nil, errors.New("redisfake: bad bulk header")
		}
		size, err := strconv.Atoi(hdr[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

func writeSimple(w *bufio.Writer, s string) { fmt.Fprintf(w, "+%s\r\n", s) }
func writeError(w *bufio.Writer, s string)  { fmt.Fprintf(w, "-%s\r\n", s) }
func writeInt(w *bufio.Writer, n int64)     { fmt.Fprintf(w, ":%d\r\n", n) }
func writeNil(w *bufio.Writer)              { fmt.Fprint(w, "$-1\r\n") }
func writeBulk(w *bufio.Writer, s string)   { fmt.Fprintf(w, "$%d\r\n%s\r\n", len(s), s) }

func (s *Server) exec(w *bufio.Writer, args []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cmds = append(s.cmds, strings.Join(args, " "))

	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "PING":
		writeSimple(w, "PONG")
	case "SELECT", "CLIENT":
		writeSimple(w, "OK")
	case "GET":
		if v, ok := s.data[args[1]]; ok {
			writeBulk(w, v)
		} else {
			writeNil(w)
		}
	case "SET":
		key, val := args[1], args[2]
		nx := false
		var ttl time.Duration
		for i := 3; i < len(args); i++ {
			switch strings.ToUpper(args[i]) {
			case "NX":
				nx = true
			case "EX":
				n, _ := strconv.Atoi(args[i+1])
				ttl = time.Duration(n) * time.Second
				i++
			case "PX":
				n, _ := strconv.Atoi(args[i+1])
				ttl = time.Duration(n) * time.Millisecond
				i++
			}
		}
		if _, exists := s.data[key]; nx && exists {
			writeNil(w)
			return
		}
		s.data[key] = val
		s.ttl[key] = ttl
		writeSimple(w, "OK")
	case "SETNX":
		if _, exists := s.data[args[1]]; exists {
			writeInt(w, 0)
			return
		}
		s.data[args[1]] = args[2]
		writeInt(w, 1)
	case "INCR", "DECR":
		n, err := strconv.ParseInt(s.dataOr(args[1], "0"), 10, 64)
		if err != nil {
			writeError(w, "ERR value is not an integer or out of range")
			return
		}
		if cmd == "INCR" {
			n++
		} else {
			n--
		}
		s.data[args[1]] = strconv.FormatInt(n, 10)
		writeInt(w, n)
	case "EXPIRE":
		if _, ok := s.data[args[1]]; !ok {
			writeInt(w, 0)
			return
		}
		n, _ := strconv.Atoi(args[2])
		s.ttl[args[1]] = time.Duration(n) * time.Second
		writeInt(w, 1)
	case "TTL":
		if _, ok := s.data[args[1]]; !ok {
			writeInt(w, -2)
			return
		}
		writeInt(w, int64(s.ttl[args[1]]/time.Second))
	case "EXISTS":
		var n int64
		for _, k := range args[1:] {
			if _, ok := s.data[k]; ok {
				n++
			}
		}
		writeInt(w, n)
	case "DEL":
		var n int64
		for _, k := range args[1:] {
			if _, ok := s.data[k]; ok {
				delete(s.data, k)
				delete(s.ttl, k)
				n++
			}
		}
		writeInt(w, n)
	case "SCAN":
		// one pass: every matching key, cursor 0
		pattern := "*"
		for i := 2; i < len(args); i++ {
			if strings.ToUpper(args[i]) == "MATCH" {
				pattern = args[i+1]
				i++
			}
		}
		var keys []string
		for k := range s.data {
			if ok, _ := path.Match(pattern, k); ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		fmt.Fprint(w, "*2\r\n")
		writeBulk(w, "0")
		fmt.Fprintf(w, "*%d\r\n", len(keys))
		for _, k := range keys {
			writeBulk(w, k)
		}
	default:
		writeError(w, "ERR unknown command '"+args[0]+"'")
	}
}

func (s *Server) dataOr(key, fallback string) string {
	if v, ok := s.data[key]; ok {
		return v
	}
	return fallback
}
