package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type ubusCall struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type LoginResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      int           `json:"id"`
	Result  []interface{} `json:"result"` // changed from []json.RawMessage to []interface{}
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type UbusClient struct {
	Host     string
	Username string
	Password string

	mu     sync.Mutex
	token  string
	expiry time.Time
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  []interface{} `json:"result"` // first element int, second element map
}

func ubusCallHostContext(ctx context.Context, st StationConfig, token string, object, method string, params interface{}) (float64, error) {
	rpcResp, err := ubusBaseCallContext(ctx, st, token, object, method, params)
	if err != nil {
		return 0, err
	}

	if len(rpcResp.Result) < 1 {
		return 0, fmt.Errorf("ubus: unexpected result length: %d", len(rpcResp.Result))
	}

	switch v := rpcResp.Result[0].(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("ubus: cannot convert json.Number to float64: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("ubus: unexpected type for result[0]: %T", v)
	}
}

func ubusCallHost(st StationConfig, token string, object, method string, params interface{}) (float64, error) {
	return ubusCallHostContext(context.Background(), st, token, object, method, params)
}

func ubusBaseCallContext(ctx context.Context, st StationConfig, token string, object string, method string, params interface{}) (JSONRPCResponse, error) {
	url := fmt.Sprintf("http://%s:%d/ubus", st.Host, st.Port)
	call := ubusCall{Jsonrpc: "2.0", ID: 1, Method: "call", Params: []interface{}{token, object, method, params}}
	b, err := json.Marshal(call)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return JSONRPCResponse{}, fmt.Errorf("ubus returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSONRPCResponse{}, fmt.Errorf("read response: %w", err)
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("invalid jsonrpc response: %w (body=%s)", err, string(body))
	}
	return rpcResp, nil
}

func ubusBaseCall(st StationConfig, token string, object string, method string, params interface{}) (JSONRPCResponse, error) {
	return ubusBaseCallContext(context.Background(), st, token, object, method, params)
}

func ubusCallHostDevicesContext(ctx context.Context, st StationConfig, token string, object, method string, params interface{}) (map[string]interface{}, error) {
	rpcResp, err := ubusBaseCallContext(ctx, st, token, object, method, params)
	if err != nil {
		return nil, err
	}
	if len(rpcResp.Result) < 2 {
		return nil, fmt.Errorf("ubus: unexpected result length: %d", len(rpcResp.Result))
	}

	devicesMapRaw, ok := rpcResp.Result[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("ubus: result[1] is not a map (got %T)", rpcResp.Result[1])
	}

	return devicesMapRaw, nil
}

func ubusCallHostDevices(st StationConfig, token string, object, method string, params interface{}) (map[string]interface{}, error) {
	return ubusCallHostDevicesContext(context.Background(), st, token, object, method, params)
}

func (c *UbusClient) ubusLoginCachedContext(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Token still valid?
	if time.Now().Before(c.expiry) && c.token != "" {
		return c.token, nil
	}

	token, ttl, err := c.ubusLoginContext(ctx)
	if err != nil {
		return "", err
	}

	// guard ttl to avoid negative or tiny expiry
	if ttl <= 10 {
		ttl = 10
	}

	c.token = token
	c.expiry = time.Now().Add(time.Duration(ttl-5) * time.Second) // 5s buffer
	return c.token, nil
}

func (c *UbusClient) ubusLoginCached() (string, error) {
	return c.ubusLoginCachedContext(context.Background())
}

func (c *UbusClient) ubusLoginContext(ctx context.Context) (token string, ttl int, err error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "call",
		"params": []interface{}{
			"00000000000000000000000000000000", // ubus socket for login
			"session",                          // object
			"login",                            // method
			map[string]string{
				"username": c.Username,
				"password": c.Password,
			},
		},
		"id": 1,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal login request: %w", err)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	url := fmt.Sprintf("http://%s/ubus", c.Host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", 0, fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("ubus login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", 0, fmt.Errorf("ubus login returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read login response: %w", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", 0, fmt.Errorf("invalid JSON response: %w, body: %s", err, string(body))
	}

	if loginResp.Error != nil {
		return "", 0, fmt.Errorf("ubus login error: %d %s", loginResp.Error.Code, loginResp.Error.Message)
	}

	// Result array: [0, token-or-details, ...] — parse without a second unmarshal
	if len(loginResp.Result) < 2 {
		return "", 0, fmt.Errorf("unexpected login result: %+v", loginResp.Result)
	}

	// parse result code (typically numeric)
	var codeInt int
	switch v := loginResp.Result[0].(type) {
	case float64:
		codeInt = int(v)
	case int:
		codeInt = v
	case int64:
		codeInt = int(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return "", 0, fmt.Errorf("invalid login result code: %w", err)
		}
		codeInt = int(n)
	default:
		return "", 0, fmt.Errorf("unexpected type for login result[0]: %T", v)
	}

	if codeInt != 0 {
		return "", 0, fmt.Errorf("login failed, code=%d", codeInt)
	}

	// parse details object (expect map[string]interface{})
	details, ok := loginResp.Result[1].(map[string]interface{})
	if !ok {
		return "", 0, fmt.Errorf("unexpected login details type: %T", loginResp.Result[1])
	}

	// token
	tokRaw, ok := details["ubus_rpc_session"]
	if !ok {
		// some devices return token as second element (string) directly; handle that too
		if s, ok := loginResp.Result[1].(string); ok && s != "" {
			return s, 0, nil
		}
		return "", 0, errors.New("ubus login: missing ubus_rpc_session")
	}
	tokenStr, ok := tokRaw.(string)
	if !ok || tokenStr == "" {
		return "", 0, fmt.Errorf("ubus login: invalid token type %T", tokRaw)
	}

	// timeout
	ttlVal := 0
	if tRaw, ok := details["timeout"]; ok {
		switch vt := tRaw.(type) {
		case float64:
			ttlVal = int(vt)
		case int:
			ttlVal = vt
		case int64:
			ttlVal = int(vt)
		case json.Number:
			n, err := vt.Int64()
			if err != nil {
				return "", 0, fmt.Errorf("invalid timeout value: %w", err)
			}
			ttlVal = int(n)
		default:
			return "", 0, fmt.Errorf("unexpected timeout type: %T", vt)
		}
	}

	if tokenStr == "" {
		return "", 0, errors.New("ubus login returned empty token")
	}

	return tokenStr, ttlVal, nil
}

func (c *UbusClient) ubusLogin() (string, int, error) {
	return c.ubusLoginContext(context.Background())
}
