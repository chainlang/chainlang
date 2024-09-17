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
	"time"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
	"hanticoin/mempool"
	"hanticoin/state"
)

const (
	validatorKeyFile = "validator_key.json"
	stateDir         = "state"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "Data directory")
	initChain := flag.Bool("init", false, "Initialize chain (genesis + state)")
	runNode := flag.Bool("run", false, "Run node (produce blocks + API)")
	keygen := flag.String("keygen", "", "Generate key and write to file (prints address)")
	sendTo := flag.String("send-to", "", "Send: recipient address (hex)")
	sendAmount := flag.Uint64("send-amount", 0, "Send: amount in smallest units")
	sendFee := flag.Uint64("send-fee", config.MinFee, "Send: fee")
	sendKey := flag.String("send-key", "", "Send: sender key file")
	sendAPI := flag.String("send-api", "http://localhost:8080", "Send: node API URL")
	sendNonce := flag.Uint64("send-nonce", 0, "Send: nonce (0 = fetch from API)")
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
	if *runNode {
		if err := runNodeLoop(*dataDir); err != nil {
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

func runNodeLoop(dataDir string) error {
	cfg := config.DefaultChainConfig()
	chain, err := core.NewChain(dataDir, cfg)
	if err != nil {
		return err
	}
	// Ensure genesis is applied
	if chain.LastHash.IsZero() {
		_, allocation, err := core.LoadGenesisFromFile(dataDir)
		if err != nil {
			return fmt.Errorf("no genesis found: run with -init first: %w", err)
		}
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
	pool := mempool.New(cfg.MaxBlockSize/500, st) // rough max txs per block
	priv, err := crypto.LoadKey(filepath.Join(dataDir, validatorKeyFile))
	if err != nil {
		return fmt.Errorf("validator key: %w", err)
	}
	validatorAddr := crypto.PubkeyToAddress(&priv.PublicKey)

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
		last, err := chain.LastBlock()
		if err != nil || last == nil {
			continue
		}
		pending := pool.Pending(100)
		if len(pending) == 0 {
			continue
		}
		txs := make([]core.Transaction, len(pending))
		for i := range pending {
			txs[i] = *pending[i]
		}
		b := core.NewBlock(last.Height+1, last.Hash, validatorAddr, txs)
		sig, err := crypto.Sign(priv, b.Hash)
		if err != nil {
			log.Printf("sign block: %v", err)
			continue
		}
		b.Signature = sig
		if err := st.ApplyBlock(b); err != nil {
			log.Printf("apply block: %v", err)
			continue
		}
		if err := chain.AppendBlock(b); err != nil {
			log.Printf("append block: %v", err)
			continue
		}
		pool.Remove(txs)
		log.Printf("block %d (%d txs)", b.Height, len(txs))
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

