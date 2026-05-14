package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/pkg/backoff"
)

// Client performs JSON-RPC calls to an execution node (optional; nil URL yields nil client from NewClient).
type Client struct {
	url        string
	apiKey     string
	authMode   string // bearer | x_api_key | authorization | token | none
	httpClient *http.Client
	maxRetries int
}

// NewClient returns nil when execution_node_url is empty or whitespace-only.
func NewClient(cfg *config.Config) *Client {
	url := strings.TrimSpace(cfg.ExecutionNodeURL)
	if url == "" {
		return nil
	}
	transport := &http.Transport{
		MaxIdleConns:        cfg.HTTP.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTP.MaxIdleConns,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	return &Client{
		url:        url,
		apiKey:     strings.TrimSpace(cfg.ExecutionAPIKey),
		authMode:   strings.TrimSpace(cfg.ExecutionAuthHeader),
		httpClient: &http.Client{Transport: transport, Timeout: cfg.HTTP.Timeout()},
		maxRetries: cfg.HTTP.MaxRetries,
	}
}

func (c *Client) applyAuth(req *http.Request) {
	mode := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.authMode), "-", "_"))
	if mode == "none" || mode == "off" || mode == "false" || mode == "disabled" {
		return
	}
	if c.apiKey == "" {
		return
	}
	switch mode {
	case "", "bearer":
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	case "x_api_key":
		req.Header.Set("x-api-key", c.apiKey)
	case "authorization", "raw":
		req.Header.Set("Authorization", c.apiKey)
	case "token":
		req.Header.Set("Token", c.apiKey)
	default:
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int    `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, Params: params, ID: 1})
	if err != nil {
		return nil, err
	}

	var lastErr error
	b := backoff.NewDefault()
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		c.applyAuth(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries && b.Wait(ctx) {
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries && b.Wait(ctx) {
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			lastErr = &backoff.RetryableError{StatusCode: resp.StatusCode, Message: string(respBody)}
			if attempt < c.maxRetries && b.Wait(ctx) {
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode != http.StatusOK {
			msg := strings.TrimSpace(string(respBody))
			if resp.StatusCode == http.StatusUnauthorized {
				if c.apiKey == "" {
					return nil, fmt.Errorf("execution rpc http 401: %s (missing or disabled auth: set execution_api_key, or use execution_auth_header none only for endpoints that do not require a token)", msg)
				}
				return nil, fmt.Errorf("execution rpc http 401: %s (check execution_api_key and execution_auth_header)", msg)
			}
			return nil, fmt.Errorf("execution rpc http %d: %s", resp.StatusCode, msg)
		}

		var out rpcResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return nil, fmt.Errorf("decode rpc response: %w", err)
		}
		if out.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
		}
		return out.Result, nil
	}

	return nil, lastErr
}

type blockForFeesJSON struct {
	BaseFeePerGas *string `json:"baseFeePerGas"`
}

type receiptJSON struct {
	GasUsed           string `json:"gasUsed"`
	EffectiveGasPrice string `json:"effectiveGasPrice"`
	GasPrice          string `json:"gasPrice"`
}

// PriorityFeesWeiDecimalString fetches execution block receipts and returns total priority fees in wei as a decimal string (no leading zeros), or error.
func (c *Client) PriorityFeesWeiDecimalString(ctx context.Context, blockNumber uint64) (string, error) {
	if c == nil {
		return "", fmt.Errorf("nil execution client")
	}
	blockParam := fmt.Sprintf("0x%x", blockNumber)

	rawBlock, err := c.call(ctx, "eth_getBlockByNumber", []any{blockParam, false})
	if err != nil {
		return "", err
	}
	if len(rawBlock) == 0 || string(rawBlock) == "null" {
		return "", fmt.Errorf("eth_getBlockByNumber: empty result for %s", blockParam)
	}

	var header blockForFeesJSON
	if err := json.Unmarshal(rawBlock, &header); err != nil {
		return "", fmt.Errorf("parse block header: %w", err)
	}

	var baseFee *big.Int
	if header.BaseFeePerGas != nil && *header.BaseFeePerGas != "" {
		baseFee, err = hexToBigInt(*header.BaseFeePerGas)
		if err != nil {
			return "", fmt.Errorf("baseFeePerGas: %w", err)
		}
	} else {
		baseFee = big.NewInt(0)
	}

	rawReceipts, err := c.call(ctx, "eth_getBlockReceipts", []any{blockParam})
	if err != nil {
		return "", err
	}
	if len(rawReceipts) == 0 || string(rawReceipts) == "null" {
		return "", fmt.Errorf("eth_getBlockReceipts: empty result for %s", blockParam)
	}

	var receipts []receiptJSON
	if err := json.Unmarshal(rawReceipts, &receipts); err != nil {
		return "", fmt.Errorf("parse receipts: %w", err)
	}

	fields := make([]ReceiptFeeFields, 0, len(receipts))
	for i := range receipts {
		r := &receipts[i]
		gu, err := hexToBigInt(r.GasUsed)
		if err != nil {
			return "", fmt.Errorf("receipt %d gasUsed: %w", i, err)
		}
		var eff *big.Int
		if r.EffectiveGasPrice != "" {
			eff, err = hexToBigInt(r.EffectiveGasPrice)
			if err != nil {
				return "", fmt.Errorf("receipt %d effectiveGasPrice: %w", i, err)
			}
		}
		var gp *big.Int
		if r.GasPrice != "" {
			gp, err = hexToBigInt(r.GasPrice)
			if err != nil {
				return "", fmt.Errorf("receipt %d gasPrice: %w", i, err)
			}
		}
		fields = append(fields, ReceiptFeeFields{
			GasUsed:           gu,
			EffectiveGasPrice: eff,
			GasPrice:          gp,
		})
	}

	sum, err := SumPriorityFeesWei(baseFee, fields)
	if err != nil {
		return "", err
	}
	return sum.String(), nil
}
