package p2p

import (
	"log"
	"net"
	"sync"
	"time"
)

// PeerSet maintains connected peers and dials seeds.
type PeerSet struct {
	mu        sync.Mutex
	peers     map[string]*peerConn
	maxPeers  int
	seeds     []string
	listenAddr string
	dialer    func(addr string) (*Conn, error)
}

type peerConn struct {
	conn *Conn
	addr string
}

// NewPeerSet creates a peer set.
func NewPeerSet(maxPeers int, seeds []string, listenAddr string) *PeerSet {
	return &PeerSet{
		peers:      make(map[string]*peerConn),
		maxPeers:   maxPeers,
		seeds:      seeds,
		listenAddr: listenAddr,
		dialer:     dialPeer,
	}
}

func dialPeer(addr string) (*Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return NewConn(conn), nil
}

// DialPeer is the default dialer (exported for use when adding peers from Hello).
func DialPeer(addr string) (*Conn, error) {
	return dialPeer(addr)
}

// TryConnect dials addr and if successful adds the peer and runs onNewPeer in a goroutine.
func (p *PeerSet) TryConnect(addr string, onNewPeer func(*Conn, string)) {
	if addr == "" || addr == p.listenAddr {
		return
	}
	p.mu.Lock()
	if len(p.peers) >= p.maxPeers {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	c, err := p.dialer(addr)
	if err != nil {
		log.Printf("p2p dial %s: %v", addr, err)
		return
	}
	remote := c.RemoteAddr().String()
	if p.addPeer(c, remote, nil) {
		go onNewPeer(c, remote)
	}
}

// ConnectToSeeds dials seeds and adds them (run once at startup).
func (p *PeerSet) ConnectToSeeds(onNewPeer func(*Conn, string)) {
	for _, seed := range p.seeds {
		if seed == "" || seed == p.listenAddr {
			continue
		}
		p.mu.Lock()
		full := len(p.peers) >= p.maxPeers
		p.mu.Unlock()
		if full {
			return
		}
		c, err := p.dialer(seed)
		if err != nil {
			log.Printf("p2p dial seed %s: %v", seed, err)
			continue
		}
		remote := c.RemoteAddr().String()
		if p.addPeer(c, remote, onNewPeer) {
			// onNewPeer runs the read loop in a goroutine
		}
	}
}

// AddPeer adds an incoming connection.
func (p *PeerSet) AddPeer(c *Conn, remoteAddr string, onNewPeer func(*Conn, string)) bool {
	return p.addPeer(c, remoteAddr, onNewPeer)
}

func (p *PeerSet) addPeer(c *Conn, remoteAddr string, onNewPeer func(*Conn, string)) bool {
	p.mu.Lock()
	if len(p.peers) >= p.maxPeers {
		p.mu.Unlock()
		c.Close()
		return false
	}
	key := remoteAddr
	if _, ok := p.peers[key]; ok {
		p.mu.Unlock()
		c.Close()
		return false
	}
	p.peers[key] = &peerConn{conn: c, addr: remoteAddr}
	p.mu.Unlock()
	if onNewPeer != nil {
		go onNewPeer(c, remoteAddr)
	}
	return true
}

// RemovePeer removes a peer by remote addr.
func (p *PeerSet) RemovePeer(remoteAddr string) {
	p.mu.Lock()
	if pc, ok := p.peers[remoteAddr]; ok {
		pc.conn.Close()
		delete(p.peers, remoteAddr)
	}
	p.mu.Unlock()
}

// Broadcast sends a message to all peers.
func (p *PeerSet) Broadcast(kind Kind, payload interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pc := range p.peers {
		go func(c *Conn) {
			if err := c.Send(kind, payload); err != nil {
				c.Close()
			}
		}(pc.conn)
	}
}

// PeerCount returns current number of peers.
func (p *PeerSet) PeerCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.peers)
}
