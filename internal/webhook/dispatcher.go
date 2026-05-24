package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

type Event struct {
	Type    string `json:"type"`    // "mint" | "burn" | "redeem" | "rebalance"
	Payload any    `json:"payload"`
}

type Dispatcher struct {
	mu     sync.RWMutex
	urls   []string
	secret []byte
}

func New(secret string) *Dispatcher {
	return &Dispatcher{secret: []byte(secret)}
}

func (d *Dispatcher) Register(url string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.urls = append(d.urls, url)
}

// Send delivers the event to all registered endpoints concurrently.
// Each request includes an X-Bolt-Signature HMAC-SHA256 header.
func (d *Dispatcher) Send(ctx context.Context, event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		slog.Error("webhook marshal", "err", err)
		return
	}

	sig := d.sign(body)

	d.mu.RLock()
	urls := make([]string, len(d.urls))
	copy(urls, d.urls)
	d.mu.RUnlock()

	for _, url := range urls {
		go func(u string) {
			if err := post(ctx, u, body, sig); err != nil {
				slog.Warn("webhook delivery failed", "url", u, "err", err)
			}
		}(url)
	}
}

func (d *Dispatcher) sign(body []byte) string {
	mac := hmac.New(sha256.New, d.secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func post(ctx context.Context, url string, body []byte, sig string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bolt-Signature", fmt.Sprintf("sha256=%s", sig))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
