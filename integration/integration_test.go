//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

var (
	apiURL = envOr("BOLT_API_URL", "http://localhost:8080")
	apiKey = envOr("BOLT_API_KEY", "dev-key")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func apiPost(t *testing.T, path string, body map[string]any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, apiURL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("POST %s decode: %v", path, err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("POST %s → %d: %v", path, resp.StatusCode, result)
	}
	return result
}

func apiGet(t *testing.T, path string) map[string]any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, apiURL+path, nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("GET %s decode: %v", path, err)
	}
	return result
}

func supplyOf(t *testing.T, symbol string) int64 {
	t.Helper()
	assets, _ := apiGet(t, "/supply")["assets"].([]any)
	for _, a := range assets {
		row, _ := a.(map[string]any)
		if row["symbol"] == symbol {
			return int64(row["total_supply"].(float64))
		}
	}
	return 0
}

func TestMintBurnUSDL(t *testing.T) {
	before := supplyOf(t, "USDL")

	apiPost(t, "/mint", map[string]any{
		"asset_symbol": "USDL",
		"amount":       1_000_000,
		"operator":     "integration-test",
	})

	after := supplyOf(t, "USDL")
	if want := before + 1_000_000; after != want {
		t.Fatalf("supply after mint: want %d, got %d", want, after)
	}

	apiPost(t, "/burn", map[string]any{
		"asset_symbol": "USDL",
		"amount":       500_000,
		"operator":     "integration-test",
	})

	final := supplyOf(t, "USDL")
	if want := after - 500_000; final != want {
		t.Fatalf("supply after burn: want %d, got %d", want, final)
	}
}

func TestBOLTMint(t *testing.T) {
	apiPost(t, "/mint", map[string]any{"asset_symbol": "USDL", "amount": 500_000, "operator": "integration-test"})
	apiPost(t, "/mint", map[string]any{"asset_symbol": "CHFL", "amount": 300_000, "operator": "integration-test"})
	apiPost(t, "/mint", map[string]any{"asset_symbol": "NOKL", "amount": 100_000, "operator": "integration-test"})

	before := supplyOf(t, "BOLT")

	apiPost(t, "/mint/bolt", map[string]any{
		"usdl":     500_000,
		"chfl":     300_000,
		"nokl":     100_000,
		"btc":      1_540,
		"operator": "integration-test",
	})

	after := supplyOf(t, "BOLT")
	if after <= before {
		t.Fatalf("BOLT supply should have increased: before=%d after=%d", before, after)
	}

	price := apiGet(t, "/price")
	navUSD, _ := price["nav_usd"].(float64)
	if navUSD <= 0 {
		t.Fatalf("nav_usd should be positive, got %f", navUSD)
	}
}

func TestReserveProof(t *testing.T) {
	proof := apiGet(t, "/reserves")
	merkleRoot, _ := proof["merkle_root"].(string)
	sig, _ := proof["signature"].(string)

	if merkleRoot == "" {
		t.Fatal("merkle_root is empty")
	}
	if sig == "" {
		t.Fatal("signature is empty")
	}

	fmt.Printf("merkle_root: %s\nsignature:   %s\n", merkleRoot, sig)
}
