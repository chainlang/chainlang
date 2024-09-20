package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
	"hanticoin/mempool"
	"hanticoin/p2p"
	"hanticoin/state"
)

const (
	validatorKeyFile = "validator_key.json"
	stateDir         = "state"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "Data directory")
	initChain := flag.Bool("init", false, "Initialize chain (genesis + state)")
	initJoin := flag.String("init-join", "", "Init from existing node's genesis (copy genesis, new key, for multi-node)")
	runNode := flag.Bool("run", false, "Run node (produce blocks + API)")
	keygen := flag.String("keygen", "", "Generate key and write to file (prints address)")
	sendTo := flag.String("send-to", "", "Send: recipient address (hex)")
	sendAmount := flag.Uint64("send-amount", 0, "Send: amount in smallest units")
	sendFee := flag.Uint64("send-fee", config.MinFee, "Send: fee")
	sendKey := flag.String("send-key", "", "Send: sender key file")
	sendAPI := flag.String("send-api", "http://localhost:8080", "Send: node API URL")
	sendNonce := flag.Uint64("send-nonce", 0, "Send: nonce (0 = fetch from API)")
	p2pListen := flag.String("p2p-listen", ":3030", "P2P listen address")
	p2pSeeds := flag.String("p2p-seeds", "", "P2P seed peers (comma-separated)")
	flag.Parse()

	if *keygen != "" {
		if err := runKeygen(*keygen); err != nil {
			log.Fatalf("keygen: %v", err)
		}
		return
	}
	if *sendTo != "" && *sendKey != "" {
		if err := runSend(*sendKey, *sendTo, *sendAmount, *sendFee, *sendNonce, *sendAPI); err != nil {
			log.Fatalf("send: %v", err)
		}
		return
	}
	if *initChain {
		if err := runInit(*dataDir); err != nil {
			log.Fatalf("init: %v", err)
		}
		fmt.Println("Initialized. Run with -run to start the node.")
		return
	}
	if *initJoin != "" {
		if err := runInitJoin(*dataDir, *initJoin); err != nil {
			log.Fatalf("init-join: %v", err)
		}
		fmt.Println("Initialized from existing genesis. Run with -run -p2p-seeds=<first node> to sync.")
		return
	}
	if *runNode {
		seeds := strings.FieldsFunc(*p2pSeeds, func(r rune) bool { return r == ',' })
		if err := runNodeLoop(*dataDir, *p2pListen, seeds); err != nil {
			log.Fatalf("run: %v", err)
		}
		return
	}
	flag.Usage()
	os.Exit(1)
}

func runInit(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	// Generate and save validator key
	priv, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	validatorAddr := crypto.PubkeyToAddress(&priv.PublicKey)
	if err := crypto.SaveKey(filepath.Join(dataDir, validatorKeyFile), priv); err != nil {
		return err
	}
	cfg := config.DefaultChainConfig()
	if err := core.WriteGenesisFile(dataDir, cfg, validatorAddr); err != nil {
		return err
	}
	st, err := state.Open(filepath.Join(dataDir, stateDir), cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	_, allocation, err := core.LoadGenesisFromFile(dataDir)
	if err != nil {
		return err
	}
	if err := st.ApplyGenesis(allocation); err != nil {
		return err
	}
	chain, err := core.NewChain(dataDir, cfg)
	if err != nil {
		return err
	}
	genesis := core.GenesisBlock(validatorAddr)
	if err := chain.AppendBlock(genesis); err != nil {
		return err
	}
	return nil
}

func runNodeLoop(dataDir, p2pListen string, seeds []string) error {
	cfg := config.DefaultChainConfig()
	chain, err := core.NewChain(dataDir, cfg)
	if err != nil {
		return err
	}
	_, allocation, err := core.LoadGenesisFromFile(dataDir)
	if err != nil {
		return fmt.Errorf("no genesis found: run with -init first: %w", err)
	}
	// Ensure genesis is applied and chain has genesis block
	if chain.LastHash.IsZero() {
		st, err := state.Open(filepath.Join(dataDir, stateDir), cfg)
		if err != nil {
			return err
		}
		if err := st.ApplyGenesis(allocation); err != nil {
			st.Close()
			return err
		}
		genesis, _, _ := core.LoadGenesisFromFile(dataDir)
		if err := chain.AppendBlock(genesis); err != nil {
			st.Close()
			return err
		}
		st.Close()
	}
	st, err := state.Open(filepath.Join(dataDir, stateDir), cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	pool := mempool.New(cfg.MaxBlockSize/500, st)
	priv, err := crypto.LoadKey(filepath.Join(dataDir, validatorKeyFile))
	if err != nil {
		return fmt.Errorf("validator key: %w", err)
	}
	validatorAddr := crypto.PubkeyToAddress(&priv.PublicKey)

	// P2P: genesis hash for hello
	genesis, _, _ := core.LoadGenesisFromFile(dataDir)
	genesisHash := genesis.Hash
	getHeight := func() uint64 {
		last, _ := chain.LastBlock()
		if last == nil {
			return 0
		}
		return last.Height
	}
	p2pSrv := p2p.NewServer(p2pListen, genesisHash, getHeight)
	peerSet := p2p.NewPeerSet(50, seeds, p2pListen)
	var blockMu sync.Mutex // serialize block apply from network and producer
	var onConn func(*p2p.Conn, string)
	onConn = func(c *p2p.Conn, remote string) {
		if !peerSet.AddPeer(c, remote, nil) {
			return
		}
		defer peerSet.RemovePeer(remote)
		for {
			kind, payload, err := c.ReadMessage()
			if err != nil {
				return
			}
			switch kind {
			case p2p.KindHello:
				var h p2p.HelloPayload
				if json.Unmarshal(payload, &h) == nil && h.ListenAddr != "" && h.ListenAddr != p2pListen {
					peerSet.TryConnect(h.ListenAddr, onConn)
				}
			case p2p.KindGetBlocks:
				var req p2p.GetBlocksPayload
				if json.Unmarshal(payload, &req) != nil {
					continue
				}
				max := req.Max
				if max <= 0 {
					max = 50
				}
				blocks, _ := chain.GetBlocksFrom(req.FromHeight, max)
				if len(blocks) > 0 {
					_ = c.Send(p2p.KindBlocks, &p2p.BlocksPayload{Blocks: blocks})
				}
			case p2p.KindBlocks:
				var body p2p.BlocksPayload
				if json.Unmarshal(payload, &body) != nil || len(body.Blocks) == 0 {
					continue
				}
				blockMu.Lock()
				last, _ := chain.LastBlock()
				// Check if first block extends our tip
				if last != nil && body.Blocks[0].PreviousHash == last.Hash {
					for _, b := range body.Blocks {
						if !b.ValidateBlock() || chain.HasBlock(b.Hash) {
							continue
						}
						cur, _ := chain.LastBlock()
						if cur != nil && b.PreviousHash != cur.Hash {
							break
						}
						_ = chain.WriteBlock(b)
						if err := st.ApplyBlock(b); err != nil {
							break
						}
						_ = chain.SetTip(b)
						pool.Remove(b.Transactions)
						peerSet.Broadcast(p2p.KindNewBlock, b)
					}
				} else if last == nil && body.Blocks[0].Height == 0 && body.Blocks[0].PreviousHash.IsZero() {
					// Sync from empty: accept chain from genesis
					for _, b := range body.Blocks {
						if !b.ValidateBlock() {
							break
						}
						cur, _ := chain.LastBlock()
						if cur != nil && b.PreviousHash != cur.Hash {
							break
						}
						if cur == nil && b.Height == 0 {
							_ = st.ApplyGenesis(allocation)
						}
						_ = chain.WriteBlock(b)
						if err := st.ApplyBlock(b); err != nil {
							break
						}
						_ = chain.SetTip(b)
						pool.Remove(b.Transactions)
					}
				} else if len(body.Blocks) > 0 {
					blocks := body.Blocks
					for i := 1; i < len(blocks); i++ {
						if blocks[i].PreviousHash != blocks[i-1].Hash || !blocks[i].ValidateBlock() {
							blocks = nil
							break
						}
					}
					if blocks != nil && blocks[0].Height == 0 && blocks[0].PreviousHash.IsZero() {
						// Full chain from genesis: reorg if longer
						if last != nil && blocks[len(blocks)-1].Height > last.Height {
							for _, b := range blocks {
								_ = chain.WriteBlock(b)
							}
							if err := st.ResetAndReplay(allocation, blocks); err == nil {
								_ = chain.SetTip(blocks[len(blocks)-1])
								for _, b := range blocks {
									pool.Remove(b.Transactions)
								}
								log.Printf("reorg to height %d", blocks[len(blocks)-1].Height)
							}
						}
					} else if blocks != nil && last != nil && blocks[len(blocks)-1].Height > last.Height {
						_ = c.Send(p2p.KindGetBlocks, &p2p.GetBlocksPayload{FromHeight: 0, Max: 10000})
					}
				}
				blockMu.Unlock()
			case p2p.KindNewBlock:
				var b core.Block
				if json.Unmarshal(payload, &b) != nil {
					continue
				}
				blockMu.Lock()
				if b.ValidateBlock() && !chain.HasBlock(b.Hash) {
					last, _ := chain.LastBlock()
					if last != nil && b.PreviousHash == last.Hash {
						_ = chain.WriteBlock(&b)
						if st.ApplyBlock(&b) == nil {
							_ = chain.SetTip(&b)
							pool.Remove(b.Transactions)
							peerSet.Broadcast(p2p.KindNewBlock, &b)
						}
					}
				}
				blockMu.Unlock()
			case p2p.KindNewTx:
				var tx core.Transaction
				if json.Unmarshal(payload, &tx) != nil {
					continue
				}
				if pool.Add(&tx) == nil {
					peerSet.Broadcast(p2p.KindNewTx, &tx)
				}
			case p2p.KindGetPeers:
				// Could send back peer list
			case p2p.KindPeers:
				var body p2p.PeersPayload
				if json.Unmarshal(payload, &body) == nil {
					for _, addr := range body.Peers {
						if addr != p2pListen && addr != "" {
							peerSet.TryConnect(addr, onConn)
						}
					}
				}
			}
		}
	}
	p2pSrv.OnConn(onConn)
	if err := p2pSrv.Listen(); err != nil {
		return fmt.Errorf("p2p listen: %w", err)
	}
	defer p2pSrv.Close()
	peerSet.ConnectToSeeds(onConn)
	// Sync: request blocks from our height+1
	go func() {
		time.Sleep(2 * time.Second)
		last, _ := chain.LastBlock()
		if last == nil {
			return
		}
		peerSet.Broadcast(p2p.KindGetBlocks, &p2p.GetBlocksPayload{FromHeight: last.Height + 1, Max: 100})
	}()

	// HTTP API: POST /tx to submit transaction
	http.HandleFunc("/tx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var tx core.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := pool.Add(&tx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		peerSet.Broadcast(p2p.KindNewTx, &tx)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "tx hash: %s\n", tx.TxHash().Hex())
	})
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		last, _ := chain.LastBlock()
		h := uint64(0)
		if last != nil {
			h = last.Height
		}
		fmt.Fprintf(w, "height: %d\nmempool: %d\n", h, pool.Size())
	})
	http.HandleFunc("/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		addrHex := r.URL.Query().Get("address")
		if addrHex == "" {
			http.Error(w, "missing address", http.StatusBadRequest)
			return
		}
		addr, err := crypto.AddressFromHex(addrHex)
		if err != nil {
			http.Error(w, "invalid address", http.StatusBadRequest)
			return
		}
		balance, _ := st.GetBalance(addr)
		nonce, _ := st.GetNonce(addr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"balance": balance,
			"nonce":   nonce,
		})
	})

	go func() {
		log.Println("API listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("API: %v", err)
		}
	}()

	// Block producer loop
	ticker := time.NewTicker(time.Duration(config.BlockTimeTargetSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		blockMu.Lock()
		last, err := chain.LastBlock()
		if err != nil || last == nil {
			blockMu.Unlock()
			continue
		}
		pending := pool.Pending(100)
		if len(pending) == 0 {
			blockMu.Unlock()
			continue
		}
		txs := make([]core.Transaction, len(pending))
		for i := range pending {
			txs[i] = *pending[i]
		}
		b := core.NewBlock(last.Height+1, last.Hash, validatorAddr, txs)
		sig, err := crypto.Sign(priv, b.Hash)
		if err != nil {
			blockMu.Unlock()
			log.Printf("sign block: %v", err)
			continue
		}
		b.Signature = sig
		if err := st.ApplyBlock(b); err != nil {
			blockMu.Unlock()
			continue
		}
		if err := chain.AppendBlock(b); err != nil {
			blockMu.Unlock()
			continue
		}
		pool.Remove(txs)
		blockMu.Unlock()
		peerSet.Broadcast(p2p.KindNewBlock, b)
		log.Printf("block %d (%d txs)", b.Height, len(txs))
	}
	return nil
}

func runInitJoin(dataDir, joinDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	genesisPath := filepath.Join(joinDir, "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("read genesis from %s: %w", joinDir, err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "genesis.json"), data, 0644); err != nil {
		return err
	}
	cfg := config.DefaultChainConfig()
	_, allocation, err := core.LoadGenesisFromFile(dataDir)
	if err != nil {
		return err
	}
	priv, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	if err := crypto.SaveKey(filepath.Join(dataDir, validatorKeyFile), priv); err != nil {
		return err
	}
	st, err := state.Open(filepath.Join(dataDir, stateDir), cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.ApplyGenesis(allocation); err != nil {
		return err
	}
	chain, err := core.NewChain(dataDir, cfg)
	if err != nil {
		return err
	}
	genesis, _, _ := core.LoadGenesisFromFile(dataDir)
	if err := chain.AppendBlock(genesis); err != nil {
		return err
	}
	return nil
}

func runKeygen(keyFile string) error {
	priv, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	if err := crypto.SaveKey(keyFile, priv); err != nil {
		return err
	}
	addr := crypto.PubkeyToAddress(&priv.PublicKey)
	fmt.Println("address:", addr.Hex())
	return nil
}

func runSend(keyFile, toAddrHex string, amount, fee, nonce uint64, apiURL string) error {
	if amount == 0 {
		return fmt.Errorf("send-amount required")
	}
	priv, err := crypto.LoadKey(keyFile)
	if err != nil {
		return err
	}
	fromAddr := crypto.PubkeyToAddress(&priv.PublicKey)
	toAddr, err := crypto.AddressFromHex(toAddrHex)
	if err != nil {
		return fmt.Errorf("invalid send-to address: %w", err)
	}
	if nonce == 0 {
		// Fetch nonce from node
		resp, err := http.Get(apiURL + "/account?address=" + fromAddr.Hex())
		if err != nil {
			return fmt.Errorf("fetch nonce: %w", err)
		}
		defer resp.Body.Close()
		var acc struct {
			Nonce uint64 `json:"nonce"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
			return fmt.Errorf("decode account: %w", err)
		}
		nonce = acc.Nonce
	}
	tx := core.Transaction{
		From:         fromAddr,
		To:           toAddr,
		Amount:       amount,
		Fee:          fee,
		Nonce:        nonce,
		SenderPubkey: crypto.MarshalPubkey(&priv.PublicKey),
	}
	hash := tx.TxHash()
	sig, err := crypto.Sign(priv, hash)
	if err != nil {
		return err
	}
	tx.Signature = sig
	body, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	resp, err := http.Post(apiURL+"/tx", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST /tx: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned %d: %s", resp.StatusCode, string(msg))
	}
	fmt.Println("tx submitted:", tx.TxHash().Hex())
	return nil
}

