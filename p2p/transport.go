package p2p

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"hanticoin/crypto"
)

const (
	readTimeout  = 30 * time.Second
	writeTimeout = 10 * time.Second
)

// Conn wraps a TCP connection with message read/write.
type Conn struct {
	net.Conn
	enc *json.Encoder
	dec *bufio.Reader
}

// NewConn creates a Conn from a net.Conn.
func NewConn(c net.Conn) *Conn {
	c.SetReadDeadline(time.Now().Add(readTimeout))
	return &Conn{
		Conn: c,
		enc:  json.NewEncoder(c),
		dec:  bufio.NewReader(c),
	}
}

// Send writes a message (one JSON line).
func (c *Conn) Send(kind Kind, payload interface{}) error {
	c.SetWriteDeadline(time.Now().Add(writeTimeout))
	data, err := EncodeMessage(kind, payload)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(append(data, '\n'))
	return err
}

// ReadMessage reads one newline-delimited JSON message.
func (c *Conn) ReadMessage() (Kind, json.RawMessage, error) {
	c.SetReadDeadline(time.Now().Add(readTimeout))
	line, err := c.dec.ReadBytes('\n')
	if err != nil {
		return "", nil, err
	}
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) == 0 {
		return "", nil, io.EOF
	}
	return DecodeMessage(line)
}

// Server runs the P2P listener and handles incoming connections.
type Server struct {
	listenAddr  string
	genesisHash crypto.Hash
	getHeight   func() uint64
	onConn      func(*Conn, string) // remote addr
	mu          sync.Mutex
	lis         net.Listener
}

// NewServer creates a P2P server.
func NewServer(listenAddr string, genesisHash crypto.Hash, getHeight func() uint64) *Server {
	return &Server{
		listenAddr:  listenAddr,
		genesisHash: genesisHash,
		getHeight:   getHeight,
	}
}

// OnConn sets the callback for each new connection (run in goroutine).
func (s *Server) OnConn(fn func(*Conn, string)) {
	s.onConn = fn
}

// Listen starts the TCP listener.
func (s *Server) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.lis = lis
	log.Printf("p2p listening on %s", s.listenAddr)
	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.lis.Accept()
		if err != nil {
			return
		}
		remote := conn.RemoteAddr().String()
		c := NewConn(conn)
		if err := c.Send(KindHello, &HelloPayload{
			GenesisHash: s.genesisHash,
			ListenAddr:  s.listenAddr,
			Height:      s.getHeight(),
		}); err != nil {
			conn.Close()
			continue
		}
		if s.onConn != nil {
			go s.onConn(c, remote)
		} else {
			conn.Close()
		}
	}
}

// Close stops the listener.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lis != nil {
		return s.lis.Close()
	}
	return nil
}
