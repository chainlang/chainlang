// Loadgen sends many signed transactions to a node to stress-test TPS.
// Usage: go run ./cmd/loadgen -api=http://localhost:8080 -key=./key.json -total=2000
// The key's account must have balance (e.g. genesis or first validator).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
)

func main() {
	api := flag.String("api", "http://localhost:8080", "Node API base URL")
	keyFile := flag.String("key", "", "Sender key file (must have balance)")
	total := flag.Int("total", 1000, "Number of txs to send")
	concurrency := flag.Int("c", 10, "Concurrent senders")
	flag.Parse()

	if *keyFile == "" {
		log.Fatal("need -key")
	}

	priv, err := crypto.LoadKey(*keyFile)
	if err != nil {
		log.Fatalf("load key: %v", err)
	}
	from := crypto.PubkeyToAddress(&priv.PublicKey)
	// Send to a burn-style address so we don't need another funded account
	var to crypto.Address
	to[19] = 1

	// Get initial nonce
	resp, err := http.Get(*api + "/account?address=" + from.Hex())
	if err != nil {
		log.Fatalf("get account: %v", err)
	}
	var acc struct {
		Nonce uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
		resp.Body.Close()
		log.Fatalf("decode account: %v", err)
	}
	resp.Body.Close()
	nonce := acc.Nonce
	amount := uint64(1)
	fee := uint64(config.MinFee)

	done := make(chan int, *total)
	start := time.Now()
	for c := 0; c < *concurrency; c++ {
		go func(offset int) {
			client := &http.Client{Timeout: 5 * time.Second}
			for i := offset; i < *total; i += *concurrency {
				n := nonce + uint64(i)
				tx := core.Transaction{
					From:         from,
					To:           to,
					Amount:       amount,
					Fee:          fee,
					Nonce:        n,
					SenderPubkey: crypto.MarshalPubkey(&priv.PublicKey),
				}
				hash := tx.TxHash()
				sig, err := crypto.Sign(priv, hash)
				if err != nil {
					continue
				}
				tx.Signature = sig
				body, _ := json.Marshal(tx)
				resp, err := client.Post(*api+"/tx", "application/json", bytes.NewReader(body))
				if err != nil {
					continue
				}
				if resp.StatusCode == http.StatusAccepted {
					done <- 1
				}
				resp.Body.Close()
			}
		}(c)
	}

	accepted := 0
	for accepted < *total {
		select {
		case <-done:
			accepted++
		case <-time.After(60 * time.Second):
			log.Printf("timeout after %d accepted", accepted)
			goto report
		}
	}
report:
	elapsed := time.Since(start).Seconds()
	tps := float64(accepted) / elapsed
	fmt.Printf("accepted: %d, elapsed: %.2fs, TPS: %.1f\n", accepted, elapsed, tps)
}
