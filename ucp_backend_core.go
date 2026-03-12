// Universal Communication Protocol - Production Server
// IP: Julius Cameron Hill
// Complete implementation with Postgres + Telegram Bridge

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

// ==================== DATABASE LAYER ====================

type Database struct {
	db *sql.DB
}

type User struct {
	ID          string                 `json:"id"`
	UniversalID string                 `json:"universal_id"`
	NativeID    string                 `json:"native_id"`
	PublicKey   string                 `json:"public_key"`
	Username    *string                `json:"username,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	LastSeen    time.Time              `json:"last_seen"`
	IsOnline    bool                   `json:"is_online"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type LinkedAccount struct {
	UserID     string    `json:"user_id"`
	Platform   string    `json:"platform"`
	AccountID  string    `json:"account_id"`
	IsActive   bool      `json:"is_active"`
	LinkedAt   time.Time `json:"linked_at"`
}

type Message struct {
	ID          string                 `json:"id"`
	MessageID   string                 `json:"message_id"`
	FromUserID  string                 `json:"from_user_id"`
	ToUserID    string                 `json:"to_user_id"`
	Platform    string                 `json:"platform"`
	Content     *string                `json:"content"`
	MediaURLs   []string               `json:"media_urls,omitempty"`
	IsEncrypted bool                   `json:"is_encrypted"`
	Signature   *string                `json:"signature,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type BridgeStatus struct {
	Platform    string     `json:"platform"`
	IsConnected bool       `json:"is_connected"`
	LastSync    *time.Time `json:"last_sync,omitempty"`
	MessageCount int64     `json:"message_count"`
	ErrorCount  int64      `json:"error_count"`
}

func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) CreateUser(ctx context.Context, u *User) error {
	metadataJSON, _ := json.Marshal(u.Metadata)
	query := `
		INSERT INTO users (universal_id, native_id, public_key, username, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, last_seen, is_online
	`
	return d.db.QueryRowContext(ctx, query, u.UniversalID, u.NativeID, u.PublicKey, u.Username, metadataJSON).
		Scan(&u.ID, &u.CreatedAt, &u.LastSeen, &u.IsOnline)
}

func (d *Database) GetUserByUniversalID(ctx context.Context, uid string) (*User, error) {
	u := &User{Metadata: make(map[string]interface{})}
	var metadataJSON []byte
	query := `SELECT id, universal_id, native_id, public_key, username, created_at, last_seen, is_online, metadata FROM users WHERE universal_id = $1`
	err := d.db.QueryRowContext(ctx, query, uid).Scan(&u.ID, &u.UniversalID, &u.NativeID, &u.PublicKey, &u.Username, &u.CreatedAt, &u.LastSeen, &u.IsOnline, &metadataJSON)
	if err != nil {
		return nil, err
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &u.Metadata)
	}
	return u, nil
}

func (d *Database) CreateMessage(ctx context.Context, m *Message) error {
	metadataJSON, _ := json.Marshal(m.Metadata)
	mediaURLsJSON, _ := json.Marshal(m.MediaURLs)
	query := `
		INSERT INTO messages (message_id, from_user_id, to_user_id, platform, content, media_urls, is_encrypted, signature, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	return d.db.QueryRowContext(ctx, query, m.MessageID, m.FromUserID, m.ToUserID, m.Platform, m.Content, mediaURLsJSON, m.IsEncrypted, m.Signature, metadataJSON).
		Scan(&m.ID, &m.CreatedAt)
}

func (d *Database) GetMessages(ctx context.Context, uid string, limit int) ([]*Message, error) {
	query := `
		SELECT id, message_id, from_user_id, to_user_id, platform, content, media_urls, is_encrypted, signature, created_at, metadata
		FROM messages WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC LIMIT $2
	`
	rows, err := d.db.QueryContext(ctx, query, uid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		m := &Message{Metadata: make(map[string]interface{})}
		var metadataJSON, mediaURLsJSON []byte
		err := rows.Scan(&m.ID, &m.MessageID, &m.FromUserID, &m.ToUserID, &m.Platform, &m.Content, &mediaURLsJSON, &m.IsEncrypted, &m.Signature, &m.CreatedAt, &metadataJSON)
		if err != nil {
			continue
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &m.Metadata)
		}
		if len(mediaURLsJSON) > 0 {
			json.Unmarshal(mediaURLsJSON, &m.MediaURLs)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (d *Database) CreateLinkedAccount(ctx context.Context, la *LinkedAccount) error {
	query := `
		INSERT INTO linked_accounts (user_id, platform, account_id, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, platform) DO UPDATE SET account_id = EXCLUDED.account_id, is_active = EXCLUDED.is_active
	`
	_, err := d.db.ExecContext(ctx, query, la.UserID, la.Platform, la.AccountID, la.IsActive)
	return err
}

func (d *Database) GetLinkedAccounts(ctx context.Context, uid string) ([]*LinkedAccount, error) {
	query := `SELECT user_id, platform, account_id, is_active, linked_at FROM linked_accounts WHERE user_id = $1 AND is_active = TRUE`
	rows, err := d.db.QueryContext(ctx, query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*LinkedAccount
	for rows.Next() {
		la := &LinkedAccount{}
		rows.Scan(&la.UserID, &la.Platform, &la.AccountID, &la.IsActive, &la.LinkedAt)
		accounts = append(accounts, la)
	}
	return accounts, nil
}

func (d *Database) UpdateBridgeStatus(ctx context.Context, bs *BridgeStatus) error {
	query := `
		UPDATE bridge_status SET is_connected = $1, last_sync = $2, message_count = $3
		WHERE platform = $4
	`
	_, err := d.db.ExecContext(ctx, query, bs.IsConnected, bs.LastSync, bs.MessageCount, bs.Platform)
	return err
}

func (d *Database) GetAllBridgeStatuses(ctx context.Context) ([]*BridgeStatus, error) {
	query := `SELECT platform, is_connected, last_sync, message_count, error_count FROM bridge_status`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*BridgeStatus
	for rows.Next() {
		bs := &BridgeStatus{}
		rows.Scan(&bs.Platform, &bs.IsConnected, &bs.LastSync, &bs.MessageCount, &bs.ErrorCount)
		statuses = append(statuses, bs)
	}
	return statuses, nil
}

func (d *Database) UpdateUserOnline(ctx context.Context, uid string, online bool) error {
	query := `UPDATE users SET is_online = $1, last_seen = NOW() WHERE id = $2`
	_, err := d.db.ExecContext(ctx, query, online, uid)
	return err
}

// ==================== TELEGRAM BRIDGE ====================

type TelegramBridge struct {
	bot          *tgbotapi.BotAPI
	db           *Database
	hub          *Hub
	isConnected  bool
	mu           sync.RWMutex
	userMappings map[int64]string
	ucpMappings  map[string]int64
}

func NewTelegramBridge(token string, db *Database, hub *Hub) (*TelegramBridge, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("✅ Telegram Bot: @%s", bot.Self.UserName)
	return &TelegramBridge{
		bot:          bot,
		db:           db,
		hub:          hub,
		userMappings: make(map[int64]string),
		ucpMappings:  make(map[string]int64),
	}, nil
}

func (tb *TelegramBridge) Start(ctx context.Context) error {
	tb.mu.Lock()
	tb.isConnected = true
	tb.mu.Unlock()

	now := time.Now()
	tb.db.UpdateBridgeStatus(ctx, &BridgeStatus{Platform: "telegram", IsConnected: true, LastSync: &now, MessageCount: 0})

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tb.bot.GetUpdatesChan(u)

	log.Println("🚀 Telegram bridge listening...")

	go func() {
		for update := range updates {
			if update.Message != nil {
				go tb.handleMessage(context.Background(), update.Message)
			}
		}
	}()

	return nil
}

func (tb *TelegramBridge) handleMessage(ctx context.Context, tgMsg *tgbotapi.Message) {
	ucpUserID, err := tb.getOrCreateUser(ctx, tgMsg.From)
	if err != nil {
		log.Printf("❌ User creation failed: %v", err)
		return
	}

	messageID := fmt.Sprintf("tg-%d-%d", tgMsg.Chat.ID, tgMsg.MessageID)
	msg := &Message{
		MessageID:  messageID,
		FromUserID: ucpUserID,
		ToUserID:   "broadcast",
		Platform:   "telegram",
		Content:    &tgMsg.Text,
		CreatedAt:  time.Unix(int64(tgMsg.Date), 0),
		Metadata: map[string]interface{}{
			"telegram_message_id": tgMsg.MessageID,
			"telegram_chat_id":    tgMsg.Chat.ID,
			"telegram_username":   tgMsg.From.UserName,
		},
	}

	if tgMsg.Photo != nil && len(tgMsg.Photo) > 0 {
		photo := tgMsg.Photo[len(tgMsg.Photo)-1]
		if fileURL, err := tb.bot.GetFileDirectURL(photo.FileID); err == nil {
			msg.MediaURLs = []string{fileURL}
		}
	}

	tb.db.CreateMessage(ctx, msg)
	tb.hub.broadcast <- msg

	log.Printf("📨 Telegram: @%s → %s", tgMsg.From.UserName, tgMsg.Text)
}

func (tb *TelegramBridge) getOrCreateUser(ctx context.Context, tgUser *tgbotapi.User) (string, error) {
	tb.mu.RLock()
	uid, exists := tb.userMappings[tgUser.ID]
	tb.mu.RUnlock()
	if exists {
		return uid, nil
	}

	universalID := fmt.Sprintf("tg-user-%d", tgUser.ID)
	nativeID := fmt.Sprintf("native-%d", time.Now().UnixNano())
	username := tgUser.UserName

	user := &User{
		UniversalID: universalID,
		NativeID:    nativeID,
		PublicKey:   base64.StdEncoding.EncodeToString([]byte("tg-key")),
		Username:    &username,
		Metadata: map[string]interface{}{
			"telegram_id": tgUser.ID,
			"telegram_username": tgUser.UserName,
		},
	}

	if err := tb.db.CreateUser(ctx, user); err != nil {
		return "", err
	}

	tb.db.CreateLinkedAccount(ctx, &LinkedAccount{
		UserID:   user.ID,
		Platform: "telegram",
		AccountID: fmt.Sprintf("%d", tgUser.ID),
		IsActive: true,
	})

	tb.mu.Lock()
	tb.userMappings[tgUser.ID] = user.ID
	tb.ucpMappings[user.ID] = tgUser.ID
	tb.mu.Unlock()

	log.Printf("👤 Created user: @%s", tgUser.UserName)
	return user.ID, nil
}

func (tb *TelegramBridge) SendMessage(ctx context.Context, msg *Message) error {
	chatID, ok := tb.ucpMappings[msg.ToUserID]
	if !ok {
		if id, ok := msg.Metadata["telegram_chat_id"].(float64); ok {
			chatID = int64(id)
		} else {
			return fmt.Errorf("no telegram mapping")
		}
	}

	if msg.Content != nil && *msg.Content != "" {
		tgMsg := tgbotapi.NewMessage(chatID, *msg.Content)
		tb.bot.Send(tgMsg)
	}

	for _, url := range msg.MediaURLs {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(url))
		tb.bot.Send(photo)
	}

	return nil
}

// ==================== WEBSOCKET HUB ====================

type Hub struct {
	clients    map[string]*Client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	db         *Database
	mu         sync.RWMutex
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

func NewHub(db *Database) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		db:         db,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()
			h.db.UpdateUserOnline(context.Background(), client.userID, true)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()
			h.db.UpdateUserOnline(context.Background(), client.userID, false)

		case message := <-h.broadcast:
			data, _ := json.Marshal(message)
			h.mu.RLock()
			if client, ok := h.clients[message.ToUserID]; ok {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client.userID)
				}
			}
			// Broadcast to all if ToUserID is "broadcast"
			if message.ToUserID == "broadcast" {
				for _, client := range h.clients {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		msg.MessageID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
		msg.FromUserID = c.userID
		msg.CreatedAt = time.Now()
		c.hub.db.CreateMessage(context.Background(), &msg)
		c.hub.broadcast <- &msg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			c.conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}

// ==================== HTTP SERVER ====================

type Server struct {
	db       *Database
	hub      *Hub
	tgBridge *TelegramBridge
	upgrader websocket.Upgrader
	crypto   *CryptoService
}

type CryptoService struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func NewCryptoService() *CryptoService {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	return &CryptoService{privateKey: priv, publicKey: pub}
}

func (c *CryptoService) GetPublicKey() string {
	return base64.StdEncoding.EncodeToString(c.publicKey)
}

func NewServer(db *Database, tgBridge *TelegramBridge) *Server {
	hub := NewHub(db)
	go hub.Run()

	return &Server{
		db:       db,
		hub:      hub,
		tgBridge: tgBridge,
		crypto:   NewCryptoService(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", 400)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{hub: s.hub, conn: conn, send: make(chan []byte, 256), userID: userID}
	s.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	uid := fmt.Sprintf("user-%d", time.Now().UnixNano())
	user := &User{
		UniversalID: uid,
		NativeID:    uid,
		PublicKey:   s.crypto.GetPublicKey(),
		Metadata:    make(map[string]interface{}),
	}
	s.db.CreateUser(r.Context(), user)
	
	// Get linked accounts
	accounts, _ := s.db.GetLinkedAccounts(r.Context(), user.ID)
	linkedMap := make(map[string]string)
	for _, acc := range accounts {
		linkedMap[acc.Platform] = acc.AccountID
	}
	
	response := map[string]interface{}{
		"universal_id": user.UniversalID,
		"native_id": user.NativeID,
		"public_key": user.PublicKey,
		"linked_accounts": linkedMap,
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("user_id")
	msgs, _ := s.db.GetMessages(r.Context(), uid, 100)
	json.NewEncoder(w).Encode(msgs)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg Message
	json.NewDecoder(r.Body).Decode(&msg)
	msg.MessageID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	msg.CreatedAt = time.Now()
	s.db.CreateMessage(r.Context(), &msg)
	
	// Route to Telegram if needed
	if msg.Platform == "telegram" && s.tgBridge != nil {
		s.tgBridge.SendMessage(r.Context(), &msg)
	}
	
	s.hub.broadcast <- &msg
	json.NewEncoder(w).Encode(msg)
}

func (s *Server) handleLinkAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string `json:"user_id"`
		Platform  string `json:"platform"`
		AccountID string `json:"account_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	la := &LinkedAccount{
		UserID:    req.UserID,
		Platform:  req.Platform,
		AccountID: req.AccountID,
		IsActive:  true,
	}
	s.db.CreateLinkedAccount(r.Context(), la)
	json.NewEncoder(w).Encode(la)
}

func (s *Server) handleGetBridges(w http.ResponseWriter, r *http.Request) {
	statuses, _ := s.db.GetAllBridgeStatuses(r.Context())
	result := make(map[string]*BridgeStatus)
	for _, st := range statuses {
		result[st.Platform] = st
	}
	json.NewEncoder(w).Encode(result)
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next(w, r)
	}
}

func main() {
	// Environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://ucp_app:change_this_password@localhost:5432/ucp?sslmode=disable"
	}
	
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tgToken == "" {
		log.Println("⚠️  TELEGRAM_BOT_TOKEN not set - Telegram bridge disabled")
	}

	// Initialize database
	db, err := NewDatabase(dbURL)
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database connected")

	// Initialize Telegram bridge
	var tgBridge *TelegramBridge
	if tgToken != "" {
		tgBridge, err = NewTelegramBridge(tgToken, db, nil)
		if err != nil {
			log.Printf("⚠️  Telegram bridge failed: %v", err)
		} else {
			tgBridge.Start(context.Background())
		}
	}

	// Initialize server
	server := NewServer(db, tgBridge)
	if tgBridge != nil {
		tgBridge.hub = server.hub
	}

	// Routes
	http.HandleFunc("/ws", server.handleWS)
	http.HandleFunc("/api/users", cors(server.handleCreateUser))
	http.HandleFunc("/api/messages", cors(server.handleGetMessages))
	http.HandleFunc("/api/messages/send", cors(server.handleSendMessage))
	http.HandleFunc("/api/accounts/link", cors(server.handleLinkAccount))
	http.HandleFunc("/api/bridges", cors(server.handleGetBridges))

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("🚀 UCP Server started on :8080")
		log.Println("📡 WebSocket: ws://localhost:8080/ws")
		log.Println("🔌 API: http://localhost:8080/api")
		log.Println("© Julius Cameron Hill")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	<-stop
	log.Println("🛑 Shutting down...")
}