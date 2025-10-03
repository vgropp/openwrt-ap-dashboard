package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	Jsonrpc string            `json:"jsonrpc"`
	Id      int               `json:"id"`
	Result  []json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type UbosClient struct {
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

func ubusCallHost(st StationConfig, token string, object, method string, params interface{}) (float64, error) {
	rpcResp, err := ubusBaseCall(st, token, object, method, params)
	if err != nil {
		return 0, err
	}
	return rpcResp.Result[0].(float64), nil
}

func ubusBaseCall(st StationConfig, token string, object string, method string, params interface{}) (JSONRPCResponse, error) {
	url := fmt.Sprintf("http://%s:%d/ubus", st.Host, st.Port)
	call := ubusCall{Jsonrpc: "2.0", ID: 1, Method: "call", Params: []interface{}{token, object, method, params}}
	b, _ := json.Marshal(call)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return JSONRPCResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return JSONRPCResponse{}, err
	}
	var rpcResp JSONRPCResponse
	if err := json.Unmarshal([]byte(body), &rpcResp); err != nil {
		return JSONRPCResponse{}, err
	}
	return rpcResp, nil
}

func ubusCallHostDevices(st StationConfig, token string, object, method string, params interface{}) (map[string]interface{}, error) {
	rpcResp, err := ubusBaseCall(st, token, object, method, params)
	if err != nil {
		return nil, err
	}
	if len(rpcResp.Result) != 2 {
		return nil, errors.New("unexpected result length")

	}

	devicesMapRaw, ok := rpcResp.Result[1].(map[string]interface{})
	if !ok {
		return nil, errors.New("result[1] is not a map")
	}

	return devicesMapRaw, nil
}

// ubusLoginCached stellt sicher, dass wir immer einen gültigen Token haben
func (c *UbosClient) ubusLoginCached() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Token noch gültig?
	if time.Now().Before(c.expiry) && c.token != "" {
		return c.token, nil
	}

	token, ttl, err := c.ubusLogin()
	if err != nil {
		return "", err
	}

	c.token = token
	c.expiry = time.Now().Add(time.Duration(ttl-5) * time.Second) // 5s Puffer
	return c.token, nil
}

// ubosLogin führt den JSON-RPC login call aus
func (c *UbosClient) ubusLogin() (token string, ttl int, err error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "call",
		"params": []interface{}{
			"00000000000000000000000000000000", // ubus socket für login
			"session",                          // object
			"login",                            // method
			map[string]string{
				"username": c.Username,
				"password": c.Password,
			},
		},
		"id": 1,
	}

	b, _ := json.Marshal(reqBody)
	resp, err := http.Post(fmt.Sprintf("http://%s/ubus", c.Host), "application/json", bytes.NewReader(b))
	if err != nil {
		return "", 0, fmt.Errorf("ubus login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", 0, fmt.Errorf("invalid JSON response: %w, body: %s", err, string(body))
	}

	if loginResp.Error != nil {
		return "", 0, fmt.Errorf("ubus login error: %d %s", loginResp.Error.Code, loginResp.Error.Message)
	}

	// Result array: [0, token, {"ubus_rpc_session": token, "timeout": ttl}]
	if len(loginResp.Result) < 2 {
		return "", 0, fmt.Errorf("unexpected login result: %+v", loginResp.Result)
	}

	var code int
	if err := json.Unmarshal(loginResp.Result[0], &code); err != nil {
		log.Fatal(err)
	}

	if code != 0 {
		log.Fatalf("login failed, code=%d", code)
	}

	var details struct {
		Token   string `json:"ubus_rpc_session"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal(loginResp.Result[1], &details); err != nil {
		log.Fatal(err)
	}

	return details.Token, details.Timeout, nil
}
