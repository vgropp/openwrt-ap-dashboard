package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]bool
}

var wsHub = &hub{conns: make(map[*websocket.Conn]bool)}

func (h *hub) broadcast(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("ws write error: %v, closing connection", err)
			c.Close()
			delete(h.conns, c)
		}
	}
}

func (h *hub) addConn(c *websocket.Conn) {
	h.mu.Lock()
	h.conns[c] = true
	h.mu.Unlock()
}

func (h *hub) removeConn(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

func broadcastWebsocket(message []byte) { wsHub.broadcast(message) }

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return
	}
	wsHub.addConn(c)
	if b, err := json.Marshal(map[string]interface{}{"clients": store.List()}); err == nil {
		c.WriteMessage(websocket.TextMessage, b)
	}
	for {
		if _, _, err := c.NextReader(); err != nil {
			wsHub.removeConn(c)
			c.Close()
			break
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func listClientsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, store.List())
}

func disconnectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mac := vars["mac"]
	if mac == "" {
		writeError(w, "mac required", http.StatusBadRequest)
		return
	}
	cl, ok := store.Get(mac)
	if !ok {
		writeError(w, "client not found", http.StatusNotFound)
		return
	}
	st := findStationByID(cl.StationID)
	if st == nil {
		writeError(w, "station not found", http.StatusInternalServerError)
		return
	}
	client := &UbosClient{
		Host:     st.Host,
		Username: st.Username,
		Password: st.Password,
	}
	token, err := client.ubusLoginCached()
	if err != nil {
		log.Printf("ubus login error for station %s: %v", st.ID, err)
		writeError(w, "auth to station failed", http.StatusBadGateway)
		return
	}
	obj := "hostapd." + cl.Iface
	params := map[string]interface{}{"addr": cl.Mac, "reason": 5, "deauth": true}
	if _, err := ubusCallHost(*st, token, obj, "del_client", params); err != nil {
		log.Printf("del_client failed for %s on %s: %v", cl.Mac, st.ID, err)
		writeError(w, "disconnect failed", http.StatusBadGateway)
		return
	}
	store.Delete(cl.Mac)
	notifyClientsUpdate()
	w.WriteHeader(http.StatusNoContent)
}

func findStationByID(id string) *StationConfig {
	if id == "" {
		return nil
	}
	if st, ok := stationMap[id]; ok {
		return st
	}
	return nil
}

func startHTTP(addr string) {
	r := mux.NewRouter()
	r.HandleFunc("/api/clients", listClientsHandler).Methods("GET")
	r.HandleFunc("/api/clients/{mac}/disconnect", disconnectHandler).Methods("POST")
	r.HandleFunc("/ws/clients", wsHandler)

	fs := http.FileServer(http.Dir("../frontend/dist"))

	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := "../frontend/dist" + req.URL.Path
		if _, err := os.Stat(path); err == nil {
			fs.ServeHTTP(w, req)
			return
		}
		http.ServeFile(w, req, "../frontend/dist/index.html")
	}))

	log.Printf("backend: listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}
