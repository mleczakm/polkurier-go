// Package polkurier is a Go client for the Polkurier WebService API (https://api.polkurier.pl), the courier
// broker behind polkurier.pl: shipment valuation, ordering parcels (InPost, DPD, DHL, UPS, Pocztex, ...),
// labels, tracking, courier pickups, parcel lockers and account data.
//
// It follows the API documentation v1.12. Every API method is a method on [Client]:
//
//	c := polkurier.New("12345", "api-token")
//	prices, err := c.OrderValuationV2(ctx, polkurier.ValuationRequest{
//		OrderRequest: polkurier.OrderRequest{
//			ShipmentType: polkurier.ShipmentBox,
//			Packs:        []polkurier.Pack{{Length: 30, Width: 20, Height: 20, Weight: 2}},
//		},
//	})
package polkurier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// API endpoints.
const (
	ProductionURL = "https://api.polkurier.pl/"
	SandboxURL    = "https://api-sandbox.polkurier.pl/"
)

// Client calls the Polkurier WebService API. It is safe for concurrent use.
type Client struct {
	login      string
	token      string
	baseURL    string
	httpClient *http.Client

	platform        string
	platformVersion string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at another endpoint, e.g. [SandboxURL] or a test server.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithSandbox uses the sandbox environment (access granted on request by api@polkurier.pl).
func WithSandbox() Option { return WithBaseURL(SandboxURL) }

// WithHTTPClient replaces the default HTTP client (30 s timeout).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithPlatform identifies the integrating application, sent as data.platform / data.platform_version
// like the official PHP SDK does.
func WithPlatform(name, version string) Option {
	return func(c *Client) { c.platform, c.platformVersion = name, version }
}

// New creates a client for the account ID (login) and API token generated in the polkurier.pl panel
// (Ustawienia → Token API).
func New(login, token string, opts ...Option) *Client {
	c := &Client{login: login, token: token, baseURL: ProductionURL, httpClient: &http.Client{Timeout: 30 * time.Second}}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is returned when the API answers with "status": "error".
type APIError struct {
	Method  string
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("polkurier %s: %s", e.Method, e.Message) }

// IsAPIError reports whether err is an error reported by the Polkurier API (as opposed to a network problem).
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

type authorization struct {
	Login string `json:"login"`
	Token string `json:"token"`
}

type envelope struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

// Call invokes any API method by name. data is marshalled as the "data" object and the "response" field of a
// successful answer is unmarshalled into out (which may be nil). Typed methods on Client are built on it; use it
// directly for methods added to the API after this package was released.
func (c *Client) Call(ctx context.Context, method string, data any, out any) error {
	payload, err := c.payload(method, data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("polkurier %s: %w", method, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20)) // labels are base64 PDFs
	if err != nil {
		return fmt.Errorf("polkurier %s: read response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("polkurier %s: HTTP %d", method, resp.StatusCode)
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("polkurier %s: invalid JSON response: %w", method, err)
	}
	if env.Status != "success" {
		return &APIError{Method: method, Message: errorMessage(env.Response)}
	}
	if out == nil || len(env.Response) == 0 || string(env.Response) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Response, out); err != nil {
		return fmt.Errorf("polkurier %s: decode response: %w", method, err)
	}
	return nil
}

func (c *Client) payload(method string, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	obj := map[string]any{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("polkurier %s: request data must be a JSON object: %w", method, err)
		}
	}
	if c.platform != "" {
		obj["platform"] = c.platform
		obj["platform_version"] = c.platformVersion
	}
	return json.Marshal(map[string]any{
		"authorization": authorization{Login: c.login, Token: c.token},
		"apimethod":     method,
		"data":          obj,
	})
}

// errorMessage extracts the human-readable error, which is usually a string but may be an object or list.
func errorMessage(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		out := list[0]
		for _, m := range list[1:] {
			out += "; " + m
		}
		return out
	}
	if len(raw) == 0 {
		return "unknown error"
	}
	return string(raw)
}
