package lnd

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
	"strconv"
	"strings"
)

// Client talks to LND's REST API with TLS and macaroon auth.
// The macaroon (a bearer credential LND uses for access control) is read from
// disk and sent as a hex header on every request.
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
		return fmt.Errorf("lnd %s %s → %d: %s", method, path, resp.StatusCode, e.Message)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// parseInt64 handles LND's habit of encoding int64 as JSON strings.
func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(strings.Trim(s, `"`), 10, 64)
	return v
}

type WalletBalance struct {
	Total       int64
	Confirmed   int64
	Unconfirmed int64
}

func (c *Client) WalletBalance(ctx context.Context) (WalletBalance, error) {
	var raw struct {
		Total       string `json:"total_balance"`
		Confirmed   string `json:"confirmed_balance"`
		Unconfirmed string `json:"unconfirmed_balance"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/balance/blockchain", nil, &raw); err != nil {
		return WalletBalance{}, err
	}
	return WalletBalance{
		Total:       parseInt64(raw.Total),
		Confirmed:   parseInt64(raw.Confirmed),
		Unconfirmed: parseInt64(raw.Unconfirmed),
	}, nil
}

type Invoice struct {
	RHash          string
	PaymentRequest string
}

func (c *Client) AddInvoice(ctx context.Context, amountSat int64, memo string) (Invoice, error) {
	var raw struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/invoices", map[string]any{
		"value": amountSat,
		"memo":  memo,
	}, &raw); err != nil {
		return Invoice{}, err
	}
	return Invoice{RHash: raw.RHash, PaymentRequest: raw.PaymentRequest}, nil
}

type Payment struct {
	PaymentHash     string
	PaymentPreimage string
	ValueSat        int64
}

func (c *Client) SendPaymentSync(ctx context.Context, paymentRequest string) (Payment, error) {
	var raw struct {
		PaymentHash     string `json:"payment_hash"`
		PaymentPreimage string `json:"payment_preimage"`
		ValueSat        string `json:"value_sat"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/channels/transactions", map[string]any{
		"payment_request": paymentRequest,
	}, &raw); err != nil {
		return Payment{}, err
	}
	return Payment{
		PaymentHash:     raw.PaymentHash,
		PaymentPreimage: raw.PaymentPreimage,
		ValueSat:        parseInt64(raw.ValueSat),
	}, nil
}

type Channel struct {
	ChannelPoint  string
	LocalBalance  int64
	RemoteBalance int64
	Active        bool
}

func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	var raw struct {
		Channels []struct {
			ChannelPoint  string `json:"channel_point"`
			LocalBalance  string `json:"local_balance"`
			RemoteBalance string `json:"remote_balance"`
			Active        bool   `json:"active"`
		} `json:"channels"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/channels", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Channel, len(raw.Channels))
	for i, ch := range raw.Channels {
		out[i] = Channel{
			ChannelPoint:  ch.ChannelPoint,
			LocalBalance:  parseInt64(ch.LocalBalance),
			RemoteBalance: parseInt64(ch.RemoteBalance),
			Active:        ch.Active,
		}
	}
	return out, nil
}
