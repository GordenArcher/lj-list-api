package repositories

import (
	"context"
	"fmt"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

// Create inserts a new message into a conversation. The caller must verify
// that the sender is a participant in the conversation, this repository
// trusts that check has already happened.
func (r *MessageRepository) Create(ctx context.Context, conversationID, senderID, content string) (*models.Message, error) {
	query := `
		INSERT INTO messages (conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, content, read_at, created_at
	`

	var msg models.Message
	err := r.pool.QueryRow(ctx, query, conversationID, senderID, content).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content,
		&msg.ReadAt, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return &msg, nil
}

// FindByConversationID returns paginated messages in a conversation, oldest
// first. The frontend inverts this for chat display (newest at bottom).
// Use with CountByConversationID to calculate pagination metadata.
func (r *MessageRepository) FindByConversationID(ctx context.Context, conversationID string, offset, limit int) ([]models.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, content, read_at, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		OFFSET $2 LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, conversationID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID,
			&msg.Content, &msg.ReadAt, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if messages == nil {
		messages = []models.Message{}
	}

	return messages, nil
}

// CountByConversationID returns the total number of messages in a conversation.
func (r *MessageRepository) CountByConversationID(ctx context.Context, conversationID string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, conversationID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages by conversation: %w", err)
	}

	return count, nil
}

// MarkAsRead sets read_at for all unread messages in a conversation that
// were not sent by the given user. Called when a user opens a conversation.
func (r *MessageRepository) MarkAsRead(ctx context.Context, conversationID, readerID string) error {
	query := `
		UPDATE messages
		SET read_at = NOW()
		WHERE conversation_id = $1 AND sender_id != $2 AND read_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, conversationID, readerID)
	if err != nil {
		return fmt.Errorf("mark messages as read: %w", err)
	}

	return nil
}
