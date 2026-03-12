// Universal Communication Protocol - Telegram Bridge
// IP: Julius Cameron Hill
// Full Telegram Bot API integration with message routing

package bridge

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramBridge handles Telegram platform integration
type TelegramBridge struct {
	bot            *tgbotapi.BotAPI
	db             *Database
	hub            *WebSocketHub
	token          string
	isConnected    bool
	mu             sync.RWMutex
	stopChan       chan struct{}
	userMappings   map[int64]string // Telegram ID -> UCP User ID
	ucpMappings    map[string]int64 // UCP User ID -> Telegram ID
	messageHandlers []MessageHandler
}

// MessageHandler processes incoming messages
type MessageHandler func(ctx context.Context, msg *Message) error

// TelegramConfig holds configuration for Telegram bridge
type TelegramConfig struct {
	BotToken string
	Debug    bool
}

// NewTelegramBridge creates a new Telegram bridge instance
func NewTelegramBridge(cfg TelegramConfig, db *Database, hub *WebSocketHub) (*TelegramBridge, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	bot.Debug = cfg.Debug

	log.Printf("Telegram bridge initialized: @%s", bot.Self.UserName)

	return &TelegramBridge{
		bot:          bot,
		db:           db,
		hub:          hub,
		token:        cfg.BotToken,
		stopChan:     make(chan struct{}),
		userMappings: make(map[int64]string),
		ucpMappings:  make(map[string]int64),
	}, nil
}

// Start begins listening for Telegram messages
func (tb *TelegramBridge) Start(ctx context.Context) error {
	tb.mu.Lock()
	if tb.isConnected {
		tb.mu.Unlock()
		return fmt.Errorf("bridge already running")
	}
	tb.isConnected = true
	tb.mu.Unlock()

	// Update bridge status
	if err := tb.updateBridgeStatus(ctx, true, nil); err != nil {
		log.Printf("Failed to update bridge status: %v", err)
	}

	// Configure update settings
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := tb.bot.GetUpdatesChan(u)

	log.Println("Telegram bridge started - listening for messages")

	go func() {
		for {
			select {
			case <-ctx.Done():
				tb.Stop(ctx)
				return
			case <-tb.stopChan:
				return
			case update := <-updates:
				if update.Message != nil {
					go tb.handleIncomingMessage(context.Background(), update.Message)
				}
			}
		}
	}()

	return nil
}

// Stop stops the Telegram bridge
func (tb *TelegramBridge) Stop(ctx context.Context) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.isConnected {
		return nil
	}

	close(tb.stopChan)
	tb.isConnected = false

	// Update bridge status
	if err := tb.updateBridgeStatus(ctx, false, nil); err != nil {
		log.Printf("Failed to update bridge status: %v", err)
	}

	log.Println("Telegram bridge stopped")
	return nil
}

// handleIncomingMessage processes incoming Telegram messages
func (tb *TelegramBridge) handleIncomingMessage(ctx context.Context, tgMsg *tgbotapi.Message) {
	// Get or create UCP user for this Telegram user
	ucpUserID, err := tb.getOrCreateUCPUser(ctx, tgMsg.From)
	if err != nil {
		log.Printf("Failed to get/create UCP user: %v", err)
		return
	}

	// Create universal message
	messageID := fmt.Sprintf("tg-%d-%d", tgMsg.Chat.ID, tgMsg.MessageID)
	
	msg := &Message{
		MessageID:  messageID,
		FromUserID: ucpUserID,
		ToUserID:   "broadcast", // Or specific user if private chat
		Platform:   "telegram",
		Content:    &tgMsg.Text,
		IsEncrypted: false,
		CreatedAt:  time.Unix(int64(tgMsg.Date), 0),
		Metadata: map[string]interface{}{
			"telegram_message_id": tgMsg.MessageID,
			"telegram_chat_id":    tgMsg.Chat.ID,
			"telegram_user_id":    tgMsg.From.ID,
			"telegram_username":   tgMsg.From.UserName,
		},
	}

	// Handle photos
	if tgMsg.Photo != nil && len(tgMsg.Photo) > 0 {
		largestPhoto := tgMsg.Photo[len(tgMsg.Photo)-1]
		fileURL, err := tb.bot.GetFileDirectURL(largestPhoto.FileID)
		if err == nil {
			msg.MediaURLs = []string{fileURL}
		}
	}

	// Handle documents
	if tgMsg.Document != nil {
		fileURL, err := tb.bot.GetFileDirectURL(tgMsg.Document.FileID)
		if err == nil {
			msg.MediaURLs = []string{fileURL}
		}
	}

	// Store message in database
	if err := tb.db.CreateMessage(ctx, msg); err != nil {
		log.Printf("Failed to store message: %v", err)
		return
	}

	// Broadcast to WebSocket clients
	tb.hub.BroadcastMessage(msg)

	// Execute message handlers
	for _, handler := range tb.messageHandlers {
		if err := handler(ctx, msg); err != nil {
			log.Printf("Message handler error: %v", err)
		}
	}

	log.Printf("Processed Telegram message from @%s: %s", tgMsg.From.UserName, tgMsg.Text)
}

// SendMessage sends a message to Telegram
func (tb *TelegramBridge) SendMessage(ctx context.Context, msg *Message) error {
	tb.mu.RLock()
	if !tb.isConnected {
		tb.mu.RUnlock()
		return fmt.Errorf("bridge not connected")
	}
	tb.mu.RUnlock()

	// Get Telegram chat ID from metadata or user mapping
	var chatID int64
	
	if msg.Metadata != nil {
		if id, ok := msg.Metadata["telegram_chat_id"].(float64); ok {
			chatID = int64(id)
		}
	}

	if chatID == 0 {
		// Try to get from user mapping
		telegramID, exists := tb.ucpMappings[msg.ToUserID]
		if !exists {
			return fmt.Errorf("no telegram mapping for user %s", msg.ToUserID)
		}
		chatID = telegramID
	}

	// Send text message
	if msg.Content != nil && *msg.Content != "" {
		tgMsg := tgbotapi.NewMessage(chatID, *msg.Content)
		
		sentMsg, err := tb.bot.Send(tgMsg)
		if err != nil {
			tb.updateBridgeStatus(ctx, true, &err.Error())
			return fmt.Errorf("failed to send telegram message: %w", err)
		}

		log.Printf("Sent message to Telegram chat %d: message_id=%d", chatID, sentMsg.MessageID)
	}

	// Send media if present
	if len(msg.MediaURLs) > 0 {
		for _, mediaURL := range msg.MediaURLs {
			photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(mediaURL))
			if _, err := tb.bot.Send(photoMsg); err != nil {
				log.Printf("Failed to send media: %v", err)
			}
		}
	}

	return nil
}

// getOrCreateUCPUser gets or creates a UCP user for a Telegram user
func (tb *TelegramBridge) getOrCreateUCPUser(ctx context.Context, tgUser *tgbotapi.User) (string, error) {
	tb.mu.RLock()
	ucpUserID, exists := tb.userMappings[tgUser.ID]
	tb.mu.RUnlock()

	if exists {
		return ucpUserID, nil
	}

	// Check if linked account exists
	// Search for user with this Telegram account
	// For now, create new user
	
	universalID := fmt.Sprintf("tg-user-%d", tgUser.ID)
	nativeID := fmt.Sprintf("native-%d", time.Now().UnixNano())
	username := tgUser.UserName

	user := &User{
		UniversalID: universalID,
		NativeID:    nativeID,
		PublicKey:   "telegram-generated-key", // Generate proper key
		Username:    &username,
		Metadata: map[string]interface{}{
			"telegram_id":         tgUser.ID,
			"telegram_username":   tgUser.UserName,
			"telegram_first_name": tgUser.FirstName,
			"telegram_last_name":  tgUser.LastName,
		},
	}

	if err := tb.db.CreateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// Create linked account
	linkedAccount := &LinkedAccount{
		UserID:          user.ID,
		Platform:        "telegram",
		AccountID:       fmt.Sprintf("%d", tgUser.ID),
		AccountUsername: &username,
		IsActive:        true,
		Credentials: map[string]interface{}{
			"telegram_id": tgUser.ID,
		},
	}

	if err := tb.db.CreateLinkedAccount(ctx, linkedAccount); err != nil {
		log.Printf("Failed to create linked account: %v", err)
	}

	// Update mappings
	tb.mu.Lock()
	tb.userMappings[tgUser.ID] = user.ID
	tb.ucpMappings[user.ID] = tgUser.ID
	tb.mu.Unlock()

	log.Printf("Created UCP user for Telegram user @%s (ID: %d)", tgUser.UserName, tgUser.ID)

	return user.ID, nil
}

// LinkTelegramAccount links an existing UCP user to a Telegram account
func (tb *TelegramBridge) LinkTelegramAccount(ctx context.Context, ucpUserID string, telegramID int64, username string) error {
	linkedAccount := &LinkedAccount{
		UserID:          ucpUserID,
		Platform:        "telegram",
		AccountID:       fmt.Sprintf("%d", telegramID),
		AccountUsername: &username,
		IsActive:        true,
		Credentials: map[string]interface{}{
			"telegram_id": telegramID,
		},
	}

	if err := tb.db.CreateLinkedAccount(ctx, linkedAccount); err != nil {
		return fmt.Errorf("failed to link account: %w", err)
	}

	// Update mappings
	tb.mu.Lock()
	tb.userMappings[telegramID] = ucpUserID
	tb.ucpMappings[ucpUserID] = telegramID
	tb.mu.Unlock()

	log.Printf("Linked Telegram account @%s to UCP user %s", username, ucpUserID)

	return nil
}

// RegisterHandler adds a message handler
func (tb *TelegramBridge) RegisterHandler(handler MessageHandler) {
	tb.messageHandlers = append(tb.messageHandlers, handler)
}

// updateBridgeStatus updates the bridge status in database
func (tb *TelegramBridge) updateBridgeStatus(ctx context.Context, connected bool, lastError *string) error {
	now := time.Now()
	status := &BridgeStatus{
		Platform:    "telegram",
		IsConnected: connected,
		LastSync:    &now,
		LastError:   lastError,
		Config: map[string]interface{}{
			"bot_username": tb.bot.Self.UserName,
		},
	}

	if lastError != nil {
		status.ErrorCount++
		status.LastErrorAt = &now
	}

	return tb.db.UpdateBridgeStatus(ctx, status)
}

// GetBotInfo returns information about the Telegram bot
func (tb *TelegramBridge) GetBotInfo() *tgbotapi.User {
	return &tb.bot.Self
}

// IsConnected returns connection status
func (tb *TelegramBridge) IsConnected() bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.isConnected
}

// SendTypingAction sends a typing indicator
func (tb *TelegramBridge) SendTypingAction(chatID int64) error {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := tb.bot.Request(action)
	return err
}

// GetChatMember gets information about a chat member
func (tb *TelegramBridge) GetChatMember(chatID int64, userID int64) (tgbotapi.ChatMember, error) {
	config := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	}
	return tb.bot.GetChatMember(config)
}

// ForwardMessage forwards a message to another chat
func (tb *TelegramBridge) ForwardMessage(fromChatID int64, toChatID int64, messageID int) error {
	forward := tgbotapi.NewForward(toChatID, fromChatID, messageID)
	_, err := tb.bot.Send(forward)
	return err
}

// DeleteMessage deletes a message
func (tb *TelegramBridge) DeleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := tb.bot.Request(deleteMsg)
	return err
}

// EditMessage edits an existing message
func (tb *TelegramBridge) EditMessage(chatID int64, messageID int, newText string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	_, err := tb.bot.Send(edit)
	return err
}

// SendPhoto sends a photo
func (tb *TelegramBridge) SendPhoto(chatID int64, photoPath string, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(photoPath))
	photo.Caption = caption
	_, err := tb.bot.Send(photo)
	return err
}

// SendDocument sends a document
func (tb *TelegramBridge) SendDocument(chatID int64, documentPath string, caption string) error {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(documentPath))
	doc.Caption = caption
	_, err := tb.bot.Send(doc)
	return err
}

// SendVoice sends a voice message
func (tb *TelegramBridge) SendVoice(chatID int64, voicePath string) error {
	voice := tgbotapi.NewVoice(chatID, tgbotapi.FilePath(voicePath))
	_, err := tb.bot.Send(voice)
	return err
}

// SendLocation sends a location
func (tb *TelegramBridge) SendLocation(chatID int64, latitude, longitude float64) error {
	location := tgbotapi.NewLocation(chatID, latitude, longitude)
	_, err := tb.bot.Send(location)
	return err
}

// CreateInlineKeyboard creates an inline keyboard
func (tb *TelegramBridge) SendMessageWithKeyboard(chatID int64, text string, buttons [][]tgbotapi.InlineKeyboardButton) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	_, err := tb.bot.Send(msg)
	return err
}