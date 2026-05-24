package tapd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client talks to tapd's REST API with TLS and macaroon auth.
type Client struct {
	base string
	mac  string
	http *http.Client
}

func New(host, tlsCertPath, macaroonPath string) (*Client, error) {
	certBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("tls cert: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certBytes)

	macBytes, err := os.ReadFile(macaroonPath)
	if err != nil {
		return nil, fmt.Errorf("macaroon: %w", err)
	}

	return &Client{
		base: "https://" + host,
		mac:  hex.EncodeToString(macBytes),
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: pool},
			},
		},
	}, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Grpc-Metadata-Macaroon", c.mac)
	if r != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("tapd %s %s → %d: %s", method, path, resp.StatusCode, e.Message)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type MintedBatch struct {
	BatchKey []byte
}

// MintAsset queues an asset in the current pending batch.
// Call FinalizeBatch to broadcast the batch transaction.
func (c *Client) MintAsset(ctx context.Context, name string, amount int64) (MintedBatch, error) {
	var raw struct {
		PendingBatch struct {
			BatchKey string `json:"batch_key"`
		} `json:"pending_batch"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/taproot-assets/mint/asset", map[string]any{
		"asset": map[string]any{
			"asset_type": 0, // NORMAL
			"name":       name,
			"amount":     amount,
		},
	}, &raw); err != nil {
		return MintedBatch{}, err
	}
	key, err := hex.DecodeString(raw.PendingBatch.BatchKey)
	if err != nil {
		return MintedBatch{}, fmt.Errorf("decode batch key: %w", err)
	}
	return MintedBatch{BatchKey: key}, nil
}

// FinalizeBatch broadcasts the pending mint batch on-chain and returns the raw proof bytes.
func (c *Client) FinalizeBatch(ctx context.Context) ([]byte, error) {
	var raw struct {
		Batch struct {
			BatchTxid string `json:"batch_txid"`
			BatchKey  string `json:"batch_key"`
		} `json:"batch"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/taproot-assets/mint/finalize", map[string]any{}, &raw); err != nil {
		return nil, err
	}
	return []byte(raw.Batch.BatchTxid), nil
}

// BurnAsset destroys supply by burning the given amount of the asset.
func (c *Client) BurnAsset(ctx context.Context, assetID []byte, amount int64) error {
	var raw json.RawMessage
	return c.do(ctx, http.MethodPost, "/v1/taproot-assets/assets/burn", map[string]any{
		"asset_id":          hex.EncodeToString(assetID),
		"amount_to_burn":    amount,
		"confirmation_text": "assets will be destroyed",
	}, &raw)
}

type TapAsset struct {
	Name    string
	AssetID []byte
	Amount  int64
}

// ListAssets returns all assets known to the tapd node.
func (c *Client) ListAssets(ctx context.Context) ([]TapAsset, error) {
	var raw struct {
		Assets []struct {
			AssetGenesis struct {
				Name    string `json:"name"`
				AssetID string `json:"asset_id"`
			} `json:"asset_genesis"`
			Amount int64 `json:"amount,string"`
		} `json:"assets"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/taproot-assets/assets", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]TapAsset, 0, len(raw.Assets))
	for _, a := range raw.Assets {
		id, _ := hex.DecodeString(a.AssetGenesis.AssetID)
		out = append(out, TapAsset{
			Name:    a.AssetGenesis.Name,
			AssetID: id,
			Amount:  a.Amount,
		})
	}
	return out, nil
}

// FetchAssetProofs retrieves the Taproot Asset proof for a given asset ID and script key.
func (c *Client) FetchAssetProofs(ctx context.Context, assetID, scriptKey []byte) ([]byte, error) {
	path := fmt.Sprintf("/v1/taproot-assets/proofs/%s/%s",
		hex.EncodeToString(assetID),
		hex.EncodeToString(scriptKey),
	)
	var raw struct {
		RawProofFile string `json:"raw_proof_file"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	return hex.DecodeString(raw.RawProofFile)
}
