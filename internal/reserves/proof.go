package reserves

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/houseofhufflepuff/bolt/internal/db"
	"github.com/houseofhufflepuff/bolt/internal/reserve"
)

type Proof struct {
	MerkleRoot []byte
	Signature  []byte
	Snapshots  []db.ReserveSnapshot
}

type Service struct {
	db      *db.DB
	reserve *reserve.Checker
	privKey ed25519.PrivateKey
}

func New(database *db.DB, checker *reserve.Checker, privKey ed25519.PrivateKey) *Service {
	if privKey == nil {
		_, privKey, _ = ed25519.GenerateKey(nil)
	}
	return &Service{db: database, reserve: checker, privKey: privKey}
}

// Snapshot records a reserve snapshot for all assets with supply > 0,
// builds a Merkle tree over the snapshots, signs the root, and persists everything.
func (s *Service) Snapshot(ctx context.Context, assets []db.Asset) (*Proof, error) {
	symbols := make([]string, 0, len(assets))
	for _, a := range assets {
		if a.TotalSupply > 0 {
			symbols = append(symbols, a.Symbol)
		}
	}
	sort.Strings(symbols)

	snaps := make([]db.ReserveSnapshot, 0, len(symbols))
	leaves := make([][]byte, 0, len(symbols))

	for _, sym := range symbols {
		asset := findAsset(assets, sym)
		bal, err := s.reserve.Balance(ctx, sym)
		if err != nil {
			return nil, fmt.Errorf("balance %s: %w", sym, err)
		}
		ratio := float64(0)
		if asset.TotalSupply > 0 {
			ratio = float64(bal) / float64(asset.TotalSupply)
		}
		snap := db.ReserveSnapshot{
			AssetSymbol:   sym,
			ReserveAmount: bal,
			SupplyAmount:  asset.TotalSupply,
			Ratio:         ratio,
		}
		snaps = append(snaps, snap)

		leaf, _ := json.Marshal(snap)
		h := sha256.Sum256(leaf)
		leaves = append(leaves, h[:])
	}

	root := merkleRoot(leaves)
	sig := ed25519.Sign(s.privKey, root)

	for _, snap := range snaps {
		snap.MerkleRoot = root
		snap.Signature = sig
		if err := s.db.InsertReserveSnapshot(ctx, snap); err != nil {
			return nil, fmt.Errorf("insert snapshot %s: %w", snap.AssetSymbol, err)
		}
	}

	return &Proof{MerkleRoot: root, Signature: sig, Snapshots: snaps}, nil
}

func merkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		z := sha256.Sum256(nil)
		return z[:]
	}
	layer := leaves
	for len(layer) > 1 {
		next := make([][]byte, 0, (len(layer)+1)/2)
		for i := 0; i < len(layer); i += 2 {
			var pair []byte
			if i+1 < len(layer) {
				pair = append(layer[i], layer[i+1]...)
			} else {
				pair = append(layer[i], layer[i]...) // duplicate last leaf
			}
			h := sha256.Sum256(pair)
			next = append(next, h[:])
		}
		layer = next
	}
	return layer[0]
}

func findAsset(assets []db.Asset, symbol string) db.Asset {
	for _, a := range assets {
		if a.Symbol == symbol {
			return a
		}
	}
	return db.Asset{}
}
