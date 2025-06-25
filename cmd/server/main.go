package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"x3dh-demo/internal/x3dh"
)

var (
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx = context.Background()
)

// ServerStats tracks server statistics
type ServerStats struct {
	TotalBundlesRegistered int64         `json:"total_bundles_registered"`
	TotalMessagesReceived  int64         `json:"total_messages_received"`
	TotalMessagesDelivered int64         `json:"total_messages_delivered"`
	ActiveBundles          int           `json:"active_bundles"`
	PendingMessages        int           `json:"pending_messages"`
	Uptime                 time.Duration `json:"uptime"`
	StartTime              time.Time     `json:"start_time"`
}

// ServerState manages the in-memory storage
type ServerState struct {
	stats    *ServerStats
}

// NewServerState creates a new server state instance
func NewServerState() *ServerState {
	state := &ServerState{
		stats: &ServerStats{
			StartTime: time.Now(),
		},
	}
	return state
}

// updateStats updates server statistics
func (s *ServerState) updateStats() {
	s.stats.Uptime = time.Since(s.stats.StartTime)
	s.stats.ActiveBundles = 0 
	s.stats.PendingMessages = 0 // Redis 
}

// GetStats returns current server statistics
func (s *ServerState) GetStats() ServerStats {
	s.updateStats()
	return *s.stats
}

// RegisterBundle registers a new bundle for a user
func (s *ServerState) RegisterBundle(userID string, bundle x3dh.Bundle) error {
	data, _ := json.Marshal(bundle)
	if err := rdb.Set(ctx, "bundle:"+userID, data, 0).Err(); err != nil {
		return err
	}
	s.stats.TotalBundlesRegistered++
	return nil
}

// GetBundle retrieves a bundle for a user
func (s *ServerState) GetBundle(userID string) (*x3dh.Bundle, bool) {
	data, err := rdb.Get(ctx, "bundle:"+userID).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		log.Printf("Warning: Failed to fetch bundle: %v", err)
		return nil, false
	}
	var bundle x3dh.Bundle
	if err := json.Unmarshal([]byte(data), &bundle); err != nil {
		log.Printf("Warning: Failed to decode bundle: %v", err)
		return nil, false
	}
	return &bundle, true
}

// StoreMessage stores a message for a user
func (s *ServerState) StoreMessage(userID string, message x3dh.InitialMessage) error {
	data, _ := json.Marshal(message)
	if err := rdb.RPush(ctx, "messages:"+userID, data).Err(); err != nil {
		return err
	}
	s.stats.TotalMessagesReceived++
	return nil
}

// GetAndDeleteMessage retrieves and deletes a message for a user
func (s *ServerState) GetAndDeleteMessage(userID string) (*x3dh.InitialMessage, int, bool) {
	data, err := rdb.LPop(ctx, "messages:"+userID).Result()
	if err == redis.Nil {
		return nil, 0, false
	} else if err != nil {
		log.Printf("Warning: Failed to fetch message: %v", err)
		return nil, 0, false
	}
	var msg x3dh.InitialMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		log.Printf("Warning: Failed to decode message: %v", err)
		return nil, 0, false
	}
	s.stats.TotalMessagesDelivered++
	return &msg, 0, true
}

// --- Global server state ---
var serverState = NewServerState()

// --- HTTP Handlers ---

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/register/")
	if user == "" {
		http.Error(w, "User not specified", http.StatusBadRequest)
		return
	}

	var bundle x3dh.Bundle
	if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		http.Error(w, "Failed to decode bundle: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := serverState.RegisterBundle(user, bundle); err != nil {
		http.Error(w, "Failed to register bundle: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func bundleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/bundle/")
	if user == "" {
		http.Error(w, "User not specified", http.StatusBadRequest)
		return
	}

	bundle, exists := serverState.GetBundle(user)
	if !exists {
		http.Error(w, "Bundle not found for user: "+user, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundle)
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/send/")
	if user == "" {
		http.Error(w, "User not specified", http.StatusBadRequest)
		return
	}
	var msg x3dh.InitialMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Failed to decode message: "+err.Error(), http.StatusBadRequest)
		return
	}
	data, _ := json.Marshal(msg)
	if err := rdb.RPush(ctx, "messages:"+user, data).Err(); err != nil {
		http.Error(w, "Failed to store message: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/messages/")
	if user == "" {
		http.Error(w, "User not specified", http.StatusBadRequest)
		return
	}
	data, err := rdb.LPop(ctx, "messages:"+user).Result()
	if err == redis.Nil {
		http.Error(w, "No new messages for user: "+user, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch message: "+err.Error(), http.StatusInternalServerError)
		return
	}
	left, _ := rdb.LLen(ctx, "messages:"+user).Result()
	var msg x3dh.InitialMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		http.Error(w, "Failed to decode message: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":      msg,
		"messages_left": left,
	}
	json.NewEncoder(w).Encode(response)
}

// statsHandler returns server statistics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	stats := serverState.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// healthHandler returns server health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    serverState.GetStats().Uptime.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// historyHandler returns all messages for a user (without deleting them)
func historyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/history/")
	if user == "" {
		http.Error(w, "User not specified", http.StatusBadRequest)
		return
	}
	data, err := rdb.LRange(ctx, "messages:"+user, 0, -1).Result()
	if err != nil {
		http.Error(w, "Failed to fetch messages: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var messages []x3dh.InitialMessage
	for _, item := range data {
		var msg x3dh.InitialMessage
		if err := json.Unmarshal([]byte(item), &msg); err == nil {
			messages = append(messages, msg)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func main() {
	http.HandleFunc("/register/", registerHandler)
	http.HandleFunc("/bundle/", bundleHandler)
	http.HandleFunc("/send/", sendMessageHandler)
	http.HandleFunc("/messages/", getMessageHandler)
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/history/", historyHandler)

	port := "8080"
	fmt.Printf("Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
} 