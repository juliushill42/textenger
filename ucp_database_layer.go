// Universal Communication Protocol - Database Layer
// IP: Julius Cameron Hill
// Production PostgreSQL implementation with connection pooling

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Database wraps sql.DB with UCP-specific methods
type Database struct {
	db *sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// User represents a UCP user
type User struct {
	ID           string                 `json:"id"`
	UniversalID  string                 `json:"universal_id"`
	NativeID     string                 `json:"native_id"`
	PublicKey    string                 `json:"public_key"`
	Username     *string                `json:"username,omitempty"`
	Email        *string                `json:"email,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastSeen     time.Time              `json:"last_seen"`
	IsOnline     bool                   `json:"is_online"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// LinkedAccount represents a platform integration
type LinkedAccount struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Platform        string                 `json:"platform"`
	AccountID       string                 `json:"account_id"`
	AccountUsername *string                `json:"account_username,omitempty"`
	Credentials     map[string]interface{} `json:"credentials"`
	IsActive        bool                   `json:"is_active"`
	LinkedAt        time.Time              `json:"linked_at"`
	LastSynced      *time.Time             `json:"last_synced,omitempty"`
}

// Message represents a communication message
type Message struct {
	ID             string                 `json:"id"`
	MessageID      string                 `json:"message_id"`
	FromUserID     string                 `json:"from_user_id"`
	ToUserID       string                 `json:"to_user_id"`
	Platform       string                 `json:"platform"`
	Content        *string                `json:"content,omitempty"`
	EncryptedData  []byte                 `json:"encrypted_data,omitempty"`
	MediaURLs      []string               `json:"media_urls,omitempty"`
	IsEncrypted    bool                   `json:"is_encrypted"`
	Signature      *string                `json:"signature,omitempty"`
	ParentMessageID *string               `json:"parent_message_id,omitempty"`
	ThreadID       *string                `json:"thread_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	DeliveredAt    *time.Time             `json:"delivered_at,omitempty"`
	ReadAt         *time.Time             `json:"read_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// BridgeStatus represents platform connectivity status
type BridgeStatus struct {
	ID          string                 `json:"id"`
	Platform    string                 `json:"platform"`
	IsConnected bool                   `json:"is_connected"`
	LastSync    *time.Time             `json:"last_sync,omitempty"`
	MessageCount int64                 `json:"message_count"`
	ErrorCount  int64                  `json:"error_count"`
	LastError   *string                `json:"last_error,omitempty"`
	LastErrorAt *time.Time             `json:"last_error_at,omitempty"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NewDatabase creates a new database connection
func NewDatabase(cfg Config) (*Database, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// ==================== USER OPERATIONS ====================

// CreateUser creates a new user
func (d *Database) CreateUser(ctx context.Context, u *User) error {
	metadataJSON, _ := json.Marshal(u.Metadata)

	query := `
		INSERT INTO users (universal_id, native_id, public_key, username, email, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, last_seen, is_online
	`

	err := d.db.QueryRowContext(
		ctx, query,
		u.UniversalID, u.NativeID, u.PublicKey, u.Username, u.Email, metadataJSON,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.LastSeen, &u.IsOnline)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (d *Database) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{Metadata: make(map[string]interface{})}
	var metadataJSON []byte

	query := `
		SELECT id, universal_id, native_id, public_key, username, email,
		       created_at, updated_at, last_seen, is_online, metadata
		FROM users
		WHERE id = $1
	`

	err := d.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.UniversalID, &u.NativeID, &u.PublicKey, &u.Username, &u.Email,
		&u.CreatedAt, &u.UpdatedAt, &u.LastSeen, &u.IsOnline, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &u.Metadata)
	}

	return u, nil
}

// GetUserByUniversalID retrieves a user by universal ID
func (d *Database) GetUserByUniversalID(ctx context.Context, universalID string) (*User, error) {
	u := &User{Metadata: make(map[string]interface{})}
	var metadataJSON []byte

	query := `
		SELECT id, universal_id, native_id, public_key, username, email,
		       created_at, updated_at, last_seen, is_online, metadata
		FROM users
		WHERE universal_id = $1
	`

	err := d.db.QueryRowContext(ctx, query, universalID).Scan(
		&u.ID, &u.UniversalID, &u.NativeID, &u.PublicKey, &u.Username, &u.Email,
		&u.CreatedAt, &u.UpdatedAt, &u.LastSeen, &u.IsOnline, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &u.Metadata)
	}

	return u, nil
}

// UpdateUserOnlineStatus updates user online status
func (d *Database) UpdateUserOnlineStatus(ctx context.Context, userID string, isOnline bool) error {
	query := `
		UPDATE users
		SET is_online = $1, last_seen = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	result, err := d.db.ExecContext(ctx, query, isOnline, userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ==================== LINKED ACCOUNT OPERATIONS ====================

// CreateLinkedAccount creates a new linked account
func (d *Database) CreateLinkedAccount(ctx context.Context, la *LinkedAccount) error {
	credentialsJSON, _ := json.Marshal(la.Credentials)

	query := `
		INSERT INTO linked_accounts (user_id, platform, account_id, account_username, credentials, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, platform) DO UPDATE
		SET account_id = EXCLUDED.account_id,
		    account_username = EXCLUDED.account_username,
		    credentials = EXCLUDED.credentials,
		    is_active = EXCLUDED.is_active,
		    linked_at = NOW()
		RETURNING id, linked_at, last_synced
	`

	err := d.db.QueryRowContext(
		ctx, query,
		la.UserID, la.Platform, la.AccountID, la.AccountUsername, credentialsJSON, la.IsActive,
	).Scan(&la.ID, &la.LinkedAt, &la.LastSynced)

	if err != nil {
		return fmt.Errorf("failed to create linked account: %w", err)
	}

	return nil
}

// GetLinkedAccounts retrieves all linked accounts for a user
func (d *Database) GetLinkedAccounts(ctx context.Context, userID string) ([]*LinkedAccount, error) {
	query := `
		SELECT id, user_id, platform, account_id, account_username, credentials,
		       is_active, linked_at, last_synced
		FROM linked_accounts
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY linked_at DESC
	`

	rows, err := d.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*LinkedAccount
	for rows.Next() {
		la := &LinkedAccount{Credentials: make(map[string]interface{})}
		var credentialsJSON []byte

		err := rows.Scan(
			&la.ID, &la.UserID, &la.Platform, &la.AccountID, &la.AccountUsername,
			&credentialsJSON, &la.IsActive, &la.LinkedAt, &la.LastSynced,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan linked account: %w", err)
		}

		if len(credentialsJSON) > 0 {
			json.Unmarshal(credentialsJSON, &la.Credentials)
		}

		accounts = append(accounts, la)
	}

	return accounts, nil
}

// GetLinkedAccountByPlatform retrieves a specific linked account
func (d *Database) GetLinkedAccountByPlatform(ctx context.Context, userID, platform string) (*LinkedAccount, error) {
	la := &LinkedAccount{Credentials: make(map[string]interface{})}
	var credentialsJSON []byte

	query := `
		SELECT id, user_id, platform, account_id, account_username, credentials,
		       is_active, linked_at, last_synced
		FROM linked_accounts
		WHERE user_id = $1 AND platform = $2
	`

	err := d.db.QueryRowContext(ctx, query, userID, platform).Scan(
		&la.ID, &la.UserID, &la.Platform, &la.AccountID, &la.AccountUsername,
		&credentialsJSON, &la.IsActive, &la.LinkedAt, &la.LastSynced,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("linked account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get linked account: %w", err)
	}

	if len(credentialsJSON) > 0 {
		json.Unmarshal(credentialsJSON, &la.Credentials)
	}

	return la, nil
}

// ==================== MESSAGE OPERATIONS ====================

// CreateMessage creates a new message
func (d *Database) CreateMessage(ctx context.Context, m *Message) error {
	metadataJSON, _ := json.Marshal(m.Metadata)
	mediaURLsJSON, _ := json.Marshal(m.MediaURLs)

	query := `
		INSERT INTO messages (
			message_id, from_user_id, to_user_id, platform, content,
			encrypted_data, media_urls, is_encrypted, signature,
			parent_message_id, thread_id, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, delivered_at, read_at
	`

	err := d.db.QueryRowContext(
		ctx, query,
		m.MessageID, m.FromUserID, m.ToUserID, m.Platform, m.Content,
		m.EncryptedData, mediaURLsJSON, m.IsEncrypted, m.Signature,
		m.ParentMessageID, m.ThreadID, metadataJSON,
	).Scan(&m.ID, &m.CreatedAt, &m.DeliveredAt, &m.ReadAt)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetMessagesByUser retrieves messages for a user
func (d *Database) GetMessagesByUser(ctx context.Context, userID string, limit int) ([]*Message, error) {
	query := `
		SELECT id, message_id, from_user_id, to_user_id, platform, content,
		       encrypted_data, media_urls, is_encrypted, signature,
		       parent_message_id, thread_id, created_at, delivered_at, read_at, metadata
		FROM messages
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := d.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		m := &Message{Metadata: make(map[string]interface{})}
		var metadataJSON, mediaURLsJSON []byte

		err := rows.Scan(
			&m.ID, &m.MessageID, &m.FromUserID, &m.ToUserID, &m.Platform, &m.Content,
			&m.EncryptedData, &mediaURLsJSON, &m.IsEncrypted, &m.Signature,
			&m.ParentMessageID, &m.ThreadID, &m.CreatedAt, &m.DeliveredAt, &m.ReadAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
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

// GetMessagesByPlatform retrieves messages by platform
func (d *Database) GetMessagesByPlatform(ctx context.Context, userID, platform string, limit int) ([]*Message, error) {
	query := `
		SELECT id, message_id, from_user_id, to_user_id, platform, content,
		       encrypted_data, media_urls, is_encrypted, signature,
		       parent_message_id, thread_id, created_at, delivered_at, read_at, metadata
		FROM messages
		WHERE (from_user_id = $1 OR to_user_id = $1) AND platform = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := d.db.QueryContext(ctx, query, userID, platform, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		m := &Message{Metadata: make(map[string]interface{})}
		var metadataJSON, mediaURLsJSON []byte

		err := rows.Scan(
			&m.ID, &m.MessageID, &m.FromUserID, &m.ToUserID, &m.Platform, &m.Content,
			&m.EncryptedData, &mediaURLsJSON, &m.IsEncrypted, &m.Signature,
			&m.ParentMessageID, &m.ThreadID, &m.CreatedAt, &m.DeliveredAt, &m.ReadAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
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

// MarkMessageAsRead marks a message as read
func (d *Database) MarkMessageAsRead(ctx context.Context, messageID, userID string) error {
	query := `
		UPDATE messages
		SET read_at = NOW()
		WHERE message_id = $1 AND to_user_id = $2 AND read_at IS NULL
	`

	result, err := d.db.ExecContext(ctx, query, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or already read")
	}

	return nil
}

// ==================== BRIDGE STATUS OPERATIONS ====================

// UpdateBridgeStatus updates bridge connectivity status
func (d *Database) UpdateBridgeStatus(ctx context.Context, bs *BridgeStatus) error {
	configJSON, _ := json.Marshal(bs.Config)

	query := `
		UPDATE bridge_status
		SET is_connected = $1,
		    last_sync = $2,
		    message_count = $3,
		    error_count = $4,
		    last_error = $5,
		    last_error_at = $6,
		    config = $7,
		    updated_at = NOW()
		WHERE platform = $8
		RETURNING id, created_at, updated_at
	`

	err := d.db.QueryRowContext(
		ctx, query,
		bs.IsConnected, bs.LastSync, bs.MessageCount, bs.ErrorCount,
		bs.LastError, bs.LastErrorAt, configJSON, bs.Platform,
	).Scan(&bs.ID, &bs.CreatedAt, &bs.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update bridge status: %w", err)
	}

	return nil
}

// GetBridgeStatus retrieves status for a specific platform
func (d *Database) GetBridgeStatus(ctx context.Context, platform string) (*BridgeStatus, error) {
	bs := &BridgeStatus{Config: make(map[string]interface{})}
	var configJSON []byte

	query := `
		SELECT id, platform, is_connected, last_sync, message_count, error_count,
		       last_error, last_error_at, config, created_at, updated_at
		FROM bridge_status
		WHERE platform = $1
	`

	err := d.db.QueryRowContext(ctx, query, platform).Scan(
		&bs.ID, &bs.Platform, &bs.IsConnected, &bs.LastSync, &bs.MessageCount, &bs.ErrorCount,
		&bs.LastError, &bs.LastErrorAt, &configJSON, &bs.CreatedAt, &bs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bridge status not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bridge status: %w", err)
	}

	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &bs.Config)
	}

	return bs, nil
}

// GetAllBridgeStatuses retrieves all bridge statuses
func (d *Database) GetAllBridgeStatuses(ctx context.Context) ([]*BridgeStatus, error) {
	query := `
		SELECT id, platform, is_connected, last_sync, message_count, error_count,
		       last_error, last_error_at, config, created_at, updated_at
		FROM bridge_status
		ORDER BY platform
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get bridge statuses: %w", err)
	}
	defer rows.Close()

	var statuses []*BridgeStatus
	for rows.Next() {
		bs := &BridgeStatus{Config: make(map[string]interface{})}
		var configJSON []byte

		err := rows.Scan(
			&bs.ID, &bs.Platform, &bs.IsConnected, &bs.LastSync, &bs.MessageCount, &bs.ErrorCount,
			&bs.LastError, &bs.LastErrorAt, &configJSON, &bs.CreatedAt, &bs.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bridge status: %w", err)
		}

		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &bs.Config)
		}

		statuses = append(statuses, bs)
	}

	return statuses, nil
}