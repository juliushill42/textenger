-- Universal Communication Protocol - Database Schema
-- IP: Julius Cameron Hill
-- Production-grade Postgres schema with full indexing

-- ============================================================
-- MIGRATION 001: Core Tables
-- ============================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table - Core identity management
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    universal_id TEXT UNIQUE NOT NULL,
    native_id TEXT UNIQUE NOT NULL,
    public_key TEXT NOT NULL,
    username TEXT,
    email TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_online BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_users_universal_id ON users(universal_id);
CREATE INDEX idx_users_native_id ON users(native_id);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_last_seen ON users(last_seen DESC);
CREATE INDEX idx_users_online ON users(is_online) WHERE is_online = TRUE;

-- Linked accounts - Platform integrations
CREATE TABLE linked_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    account_id TEXT NOT NULL,
    account_username TEXT,
    credentials JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT TRUE,
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_synced TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, platform)
);

CREATE INDEX idx_linked_accounts_user_id ON linked_accounts(user_id);
CREATE INDEX idx_linked_accounts_platform ON linked_accounts(platform);
CREATE INDEX idx_linked_accounts_active ON linked_accounts(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_linked_accounts_account_id ON linked_accounts(platform, account_id);

-- Messages table - All communications
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id TEXT UNIQUE NOT NULL,
    from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    content TEXT,
    encrypted_data BYTEA,
    media_urls TEXT[],
    is_encrypted BOOLEAN DEFAULT FALSE,
    signature TEXT,
    parent_message_id UUID REFERENCES messages(id),
    thread_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_messages_from_user ON messages(from_user_id, created_at DESC);
CREATE INDEX idx_messages_to_user ON messages(to_user_id, created_at DESC);
CREATE INDEX idx_messages_platform ON messages(platform, created_at DESC);
CREATE INDEX idx_messages_thread ON messages(thread_id, created_at ASC) WHERE thread_id IS NOT NULL;
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_unread ON messages(to_user_id, read_at) WHERE read_at IS NULL;
CREATE INDEX idx_messages_message_id ON messages(message_id);

-- Conversations table - Chat threads
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id TEXT UNIQUE NOT NULL,
    name TEXT,
    platform TEXT NOT NULL,
    is_group BOOLEAN DEFAULT FALSE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_conversations_platform ON conversations(platform);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);
CREATE INDEX idx_conversations_last_message ON conversations(last_message_at DESC);

-- Conversation participants
CREATE TABLE conversation_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    last_read_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(conversation_id, user_id)
);

CREATE INDEX idx_conv_participants_conv_id ON conversation_participants(conversation_id);
CREATE INDEX idx_conv_participants_user_id ON conversation_participants(user_id);
CREATE INDEX idx_conv_participants_active ON conversation_participants(is_active) WHERE is_active = TRUE;

-- Bridge status - Platform connectivity tracking
CREATE TABLE bridge_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    platform TEXT UNIQUE NOT NULL,
    is_connected BOOLEAN DEFAULT FALSE,
    last_sync TIMESTAMP WITH TIME ZONE,
    message_count BIGINT DEFAULT 0,
    error_count BIGINT DEFAULT 0,
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    config JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_bridge_status_platform ON bridge_status(platform);
CREATE INDEX idx_bridge_status_connected ON bridge_status(is_connected);

-- Platform credentials - Secure storage
CREATE TABLE platform_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    credential_type TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, platform, credential_type)
);

CREATE INDEX idx_platform_creds_user_platform ON platform_credentials(user_id, platform);

-- Encryption keys - User key management
CREATE TABLE encryption_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_type TEXT NOT NULL,
    public_key TEXT NOT NULL,
    private_key_encrypted BYTEA,
    algorithm TEXT NOT NULL DEFAULT 'ed25519',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(user_id, key_type, is_active) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_encryption_keys_user_id ON encryption_keys(user_id);
CREATE INDEX idx_encryption_keys_active ON encryption_keys(is_active) WHERE is_active = TRUE;

-- Audit log - Security and compliance
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    platform TEXT,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user_id ON audit_log(user_id, created_at DESC);
CREATE INDEX idx_audit_log_action ON audit_log(action, created_at DESC);
CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
CREATE INDEX idx_audit_log_platform ON audit_log(platform) WHERE platform IS NOT NULL;

-- Message queue - Async processing
CREATE TABLE message_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    action TEXT NOT NULL,
    payload JSONB NOT NULL,
    priority INTEGER DEFAULT 5,
    status TEXT DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_message_queue_status ON message_queue(status, priority DESC, created_at ASC) WHERE status = 'pending';
CREATE INDEX idx_message_queue_retry ON message_queue(next_retry_at) WHERE status = 'failed' AND retry_count < max_retries;
CREATE INDEX idx_message_queue_platform ON message_queue(platform, status);

-- Statistics table - Analytics
CREATE TABLE statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name TEXT NOT NULL,
    metric_value NUMERIC NOT NULL,
    dimensions JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_statistics_metric ON statistics(metric_name, recorded_at DESC);
CREATE INDEX idx_statistics_dimensions ON statistics USING gin(dimensions);

-- ============================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER linked_accounts_updated_at BEFORE UPDATE ON linked_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER conversations_updated_at BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER bridge_status_updated_at BEFORE UPDATE ON bridge_status
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Update conversation last_message_at on new message
CREATE OR REPLACE FUNCTION update_conversation_last_message()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE conversations
    SET last_message_at = NEW.created_at,
        updated_at = NOW()
    WHERE conversation_id = NEW.thread_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER messages_update_conversation AFTER INSERT ON messages
    FOR EACH ROW WHEN (NEW.thread_id IS NOT NULL)
    EXECUTE FUNCTION update_conversation_last_message();

-- Increment bridge message count
CREATE OR REPLACE FUNCTION increment_bridge_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE bridge_status
    SET message_count = message_count + 1,
        last_sync = NOW(),
        updated_at = NOW()
    WHERE platform = NEW.platform;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER messages_increment_bridge_count AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION increment_bridge_message_count();

-- ============================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================

-- Active conversations with unread counts
CREATE OR REPLACE VIEW user_conversations AS
SELECT 
    cp.user_id,
    c.id as conversation_id,
    c.conversation_id as thread_id,
    c.name,
    c.platform,
    c.is_group,
    c.last_message_at,
    COUNT(m.id) FILTER (WHERE m.read_at IS NULL AND m.to_user_id = cp.user_id) as unread_count,
    MAX(m.created_at) as last_message_time
FROM conversation_participants cp
JOIN conversations c ON c.id = cp.conversation_id
LEFT JOIN messages m ON m.thread_id = c.conversation_id
WHERE cp.is_active = TRUE
GROUP BY cp.user_id, c.id, c.conversation_id, c.name, c.platform, c.is_group, c.last_message_at;

-- Bridge health summary
CREATE OR REPLACE VIEW bridge_health AS
SELECT 
    platform,
    is_connected,
    message_count,
    error_count,
    CASE 
        WHEN error_count = 0 THEN 'healthy'
        WHEN error_count < 10 THEN 'degraded'
        ELSE 'unhealthy'
    END as health_status,
    last_sync,
    last_error_at
FROM bridge_status;

-- User activity summary
CREATE OR REPLACE VIEW user_activity AS
SELECT 
    u.id as user_id,
    u.username,
    u.is_online,
    u.last_seen,
    COUNT(DISTINCT la.platform) as linked_platforms,
    COUNT(DISTINCT m_sent.id) as messages_sent,
    COUNT(DISTINCT m_received.id) as messages_received
FROM users u
LEFT JOIN linked_accounts la ON la.user_id = u.id AND la.is_active = TRUE
LEFT JOIN messages m_sent ON m_sent.from_user_id = u.id AND m_sent.created_at > NOW() - INTERVAL '24 hours'
LEFT JOIN messages m_received ON m_received.to_user_id = u.id AND m_received.created_at > NOW() - INTERVAL '24 hours'
GROUP BY u.id, u.username, u.is_online, u.last_seen;

-- ============================================================
-- INITIAL DATA
-- ============================================================

-- Insert default bridge status for all platforms
INSERT INTO bridge_status (platform, is_connected, message_count) VALUES
    ('native', TRUE, 0),
    ('telegram', FALSE, 0),
    ('discord', FALSE, 0),
    ('whatsapp', FALSE, 0),
    ('meet', FALSE, 0),
    ('zoom', FALSE, 0),
    ('messenger', FALSE, 0)
ON CONFLICT (platform) DO NOTHING;

-- ============================================================
-- GRANTS (Adjust based on your security requirements)
-- ============================================================

-- Create application role
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ucp_app') THEN
        CREATE ROLE ucp_app WITH LOGIN PASSWORD 'change_this_password';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE postgres TO ucp_app;
GRANT USAGE ON SCHEMA public TO ucp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ucp_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ucp_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO ucp_app;

-- ============================================================
-- CLEANUP FUNCTIONS
-- ============================================================

-- Clean old audit logs (run periodically)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_log
    WHERE created_at < NOW() - (days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Clean processed message queue items
CREATE OR REPLACE FUNCTION cleanup_processed_queue(days_to_keep INTEGER DEFAULT 7)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM message_queue
    WHERE status IN ('processed', 'failed')
    AND processed_at < NOW() - (days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================

-- Composite indexes for common query patterns
CREATE INDEX idx_messages_user_platform_time ON messages(to_user_id, platform, created_at DESC);
CREATE INDEX idx_messages_conversation_time ON messages(thread_id, created_at ASC) WHERE thread_id IS NOT NULL;
CREATE INDEX idx_audit_user_action_time ON audit_log(user_id, action, created_at DESC);

-- Partial indexes for active records
CREATE INDEX idx_active_conversations ON conversation_participants(user_id, conversation_id) WHERE is_active = TRUE;
CREATE INDEX idx_unread_messages ON messages(to_user_id) WHERE read_at IS NULL;

-- GIN indexes for JSONB columns
CREATE INDEX idx_messages_metadata ON messages USING gin(metadata);
CREATE INDEX idx_bridge_config ON bridge_status USING gin(config);
CREATE INDEX idx_user_metadata ON users USING gin(metadata);

COMMENT ON TABLE users IS 'Core user identity and profile information';
COMMENT ON TABLE messages IS 'All messages across all platforms with encryption support';
COMMENT ON TABLE bridge_status IS 'Real-time status of platform bridges';
COMMENT ON TABLE audit_log IS 'Security audit trail for compliance';
COMMENT ON TABLE message_queue IS 'Async message processing queue with retry logic';