#!/usr/bin/env bash
# Create 5 testnet node dirs. First node is inited; others join from node1.
# Usage: ./scripts/testnet-bootstrap.sh [path_to_hanticoin_binary]
set -e
BIN="${1:-./hanticoin}"
DIR="${TESTNET_DIR:-./testnet}"
mkdir -p "$DIR"
cd "$DIR"

echo "Bootstrapping testnet under $DIR (binary: $BIN)"

$BIN -init -data-dir=./node1
echo "Node1 inited. Start with: $BIN -run -data-dir=./node1 -p2p-listen=127.0.0.1:3030"

for i in 2 3 4 5; do
  $BIN -init-join=./node1 -data-dir=./node$i
  SEEDS="127.0.0.1:3030"
  [ $i -gt 2 ] && SEEDS="127.0.0.1:3030,127.0.0.1:3031"
  PORT=$((3029 + i))
  echo "Node$i inited. Start with: $BIN -run -data-dir=./node$i -p2p-listen=127.0.0.1:$PORT -p2p-seeds=$SEEDS"
done

echo "Done. Start node1 first, then node2..5 in other terminals. See docs/TESTNET.md"
