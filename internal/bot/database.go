package bot

import (
	"database/sql"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// initDatabase initializes the database schema
func (b *Bot) initDatabase() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			telegram_id BIGINT UNIQUE NOT NULL,
			username VARCHAR(255),
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS servers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			secret_key VARCHAR(64) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			owner_id BIGINT REFERENCES users(telegram_id),
			last_seen TIMESTAMP,
			status VARCHAR(20) DEFAULT 'offline',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS user_servers (
			user_id BIGINT REFERENCES users(telegram_id),
			server_id UUID REFERENCES servers(id),
			role VARCHAR(20) DEFAULT 'viewer',
			created_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (user_id, server_id)
		)`,
		
		`CREATE TABLE IF NOT EXISTS command_history (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT REFERENCES users(telegram_id),
			server_id UUID REFERENCES servers(id),
			command VARCHAR(100) NOT NULL,
			response JSONB,
			executed_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_servers_secret_key ON servers(secret_key)`,
		`CREATE INDEX IF NOT EXISTS idx_servers_owner_id ON servers(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_servers_user_id ON user_servers(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_command_history_server_id ON command_history(server_id)`,
	}

	for _, query := range queries {
		if _, err := b.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	b.logger.Info("Database schema initialized successfully")
	return nil
}

// registerUser registers a new user or updates existing one
func (b *Bot) registerUser(user *tgbotapi.User) error {
	query := `
		INSERT INTO users (telegram_id, username, first_name, last_name, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (telegram_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = NOW()
	`

	_, err := b.db.Exec(query, user.ID, user.UserName, user.FirstName, user.LastName)
	if err != nil {
		return fmt.Errorf("failed to register user: %v", err)
	}

	b.logger.WithField("user_id", user.ID).Info("User registered/updated")
	return nil
}

// connectServer connects a server to a user
func (b *Bot) connectServer(userID int64, serverKey string) error {
	tx, err := b.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if server exists
	var serverID string
	var serverName string
	err = tx.QueryRow(`
		SELECT id, name FROM servers WHERE secret_key = $1
	`, serverKey).Scan(&serverID, &serverName)
	
	if err == sql.ErrNoRows {
		// Create new server entry
		err = tx.QueryRow(`
			INSERT INTO servers (secret_key, name, description, owner_id, status)
			VALUES ($1, $2, $3, $4, 'offline')
			RETURNING id
		`, serverKey, "New Server", "ServerEye monitored server", userID).Scan(&serverID)
		
		if err != nil {
			return fmt.Errorf("failed to create server: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query server: %v", err)
	}

	// Connect user to server
	_, err = tx.Exec(`
		INSERT INTO user_servers (user_id, server_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (user_id, server_id) DO NOTHING
	`, userID, serverID)
	
	if err != nil {
		return fmt.Errorf("failed to connect user to server: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	b.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"server_key": serverKey[:12] + "...",
	}).Info("Server connected to user")

	return nil
}

// getUserServers returns list of server keys for a user
func (b *Bot) getUserServers(userID int64) ([]string, error) {
	query := `
		SELECT s.secret_key 
		FROM servers s
		JOIN user_servers us ON s.id = us.server_id
		WHERE us.user_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := b.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user servers: %v", err)
	}
	defer rows.Close()

	var servers []string
	for rows.Next() {
		var serverKey string
		if err := rows.Scan(&serverKey); err != nil {
			return nil, fmt.Errorf("failed to scan server key: %v", err)
		}
		servers = append(servers, serverKey)
	}

	return servers, nil
}

// updateServerStatus updates the last seen timestamp for a server
func (b *Bot) updateServerStatus(serverKey string, status string) error {
	query := `
		UPDATE servers 
		SET status = $1, last_seen = NOW(), updated_at = NOW()
		WHERE secret_key = $2
	`

	_, err := b.db.Exec(query, status, serverKey)
	if err != nil {
		return fmt.Errorf("failed to update server status: %v", err)
	}

	return nil
}

// logCommand logs a command execution
func (b *Bot) logCommand(userID int64, serverKey, command string, response interface{}) error {
	// Get server ID
	var serverID string
	err := b.db.QueryRow(`
		SELECT id FROM servers WHERE secret_key = $1
	`, serverKey).Scan(&serverID)
	
	if err != nil {
		return fmt.Errorf("failed to get server ID: %v", err)
	}

	// Log command
	query := `
		INSERT INTO command_history (user_id, server_id, command, response)
		VALUES ($1, $2, $3, $4)
	`

	_, err = b.db.Exec(query, userID, serverID, command, response)
	if err != nil {
		return fmt.Errorf("failed to log command: %v", err)
	}

	return nil
}

// renameServer updates server name in database
func (b *Bot) renameServer(serverKey, newName string) error {
	query := `
		UPDATE servers 
		SET name = $1, updated_at = NOW()
		WHERE secret_key = $2
	`
	
	_, err := b.db.Exec(query, newName, serverKey)
	return err
}

// removeServer removes server and user association
func (b *Bot) removeServer(userID int64, serverKey string) error {
	tx, err := b.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Remove user-server association
	_, err = tx.Exec(`
		DELETE FROM user_servers 
		WHERE user_id = $1 AND server_id = (
			SELECT id FROM servers WHERE secret_key = $2
		)
	`, userID, serverKey)
	if err != nil {
		return err
	}
	
	// Check if server has other users
	var userCount int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM user_servers us
		JOIN servers s ON us.server_id = s.id
		WHERE s.secret_key = $1
	`, serverKey).Scan(&userCount)
	if err != nil {
		return err
	}
	
	// If no other users, delete server completely
	if userCount == 0 {
		_, err = tx.Exec(`DELETE FROM servers WHERE secret_key = $1`, serverKey)
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

// ServerInfo represents server information
type ServerInfo struct {
	SecretKey string
	Name      string
	Status    string
}

// getUserServersWithInfo returns list of servers with names for a user
func (b *Bot) getUserServersWithInfo(userID int64) ([]ServerInfo, error) {
	query := `
		SELECT s.secret_key, s.name, s.status
		FROM servers s
		JOIN user_servers us ON s.id = us.server_id
		WHERE us.user_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := b.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []ServerInfo
	for rows.Next() {
		var server ServerInfo
		if err := rows.Scan(&server.SecretKey, &server.Name, &server.Status); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}
