package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationRepository struct {
	pool *pgxpool.Pool
}

func NewConversationRepository(pool *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{pool: pool}
}

type conversationQueryRow interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// FindOrCreate upserts a conversation pair and returns the canonical row.
// The unique participant constraint guarantees only one conversation per pair.
func (r *ConversationRepository) FindOrCreate(ctx context.Context, userOne, userTwo string) (*models.Conversation, error) {
	first, second := sortPair(userOne, userTwo)
	conv, err := upsertConversation(ctx, r.pool, first, second)
	if err != nil {
		return nil, fmt.Errorf("upsert conversation: %w", err)
	}
	return conv, nil
}

// FindOrCreateWithInitialMessage wraps conversation creation + first message in
// one transaction so we don't create empty conversations on message failure.
// If the conversation already exists, no new message is inserted. The bool
// return indicates whether a new conversation (and therefore the initial
// message) was actually created.
func (r *ConversationRepository) FindOrCreateWithInitialMessage(ctx context.Context, userOne, userTwo, senderID, initialMessage string) (*models.Conversation, bool, error) {
	first, second := sortPair(userOne, userTwo)
	trimmed := strings.TrimSpace(initialMessage)

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("begin conversation transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	conv, created, err := insertConversationIfMissing(ctx, tx, first, second)
	if err != nil {
		return nil, false, fmt.Errorf("insert conversation if missing: %w", err)
	}
	if !created {
		conv, err = findConversationByParticipants(ctx, tx, first, second)
		if err != nil {
			return nil, false, fmt.Errorf("find existing conversation: %w", err)
		}
	}

	if created && trimmed != "" {
		query := `
			INSERT INTO messages (conversation_id, sender_id, content)
			VALUES ($1, $2, $3)
		`
		if _, err := tx.Exec(ctx, query, conv.ID, senderID, trimmed); err != nil {
			return nil, false, fmt.Errorf("insert initial message: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit conversation transaction: %w", err)
	}

	return conv, created, nil
}

func upsertConversation(ctx context.Context, q conversationQueryRow, first, second string) (*models.Conversation, error) {
	query := `
		INSERT INTO conversations (participant_one, participant_two)
		VALUES ($1, $2)
		ON CONFLICT (participant_one, participant_two)
		DO UPDATE SET participant_one = conversations.participant_one
		RETURNING id, participant_one, participant_two, created_at
	`

	var conv models.Conversation
	err := q.QueryRow(ctx, query, first, second).Scan(
		&conv.ID, &conv.ParticipantOne, &conv.ParticipantTwo, &conv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func insertConversationIfMissing(ctx context.Context, q conversationQueryRow, first, second string) (*models.Conversation, bool, error) {
	query := `
		INSERT INTO conversations (participant_one, participant_two)
		VALUES ($1, $2)
		ON CONFLICT (participant_one, participant_two) DO NOTHING
		RETURNING id, participant_one, participant_two, created_at
	`

	var conv models.Conversation
	err := q.QueryRow(ctx, query, first, second).Scan(
		&conv.ID, &conv.ParticipantOne, &conv.ParticipantTwo, &conv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &conv, true, nil
}

func findConversationByParticipants(ctx context.Context, q conversationQueryRow, first, second string) (*models.Conversation, error) {
	query := `
		SELECT id, participant_one, participant_two, created_at
		FROM conversations
		WHERE participant_one = $1 AND participant_two = $2
	`

	var conv models.Conversation
	err := q.QueryRow(ctx, query, first, second).Scan(
		&conv.ID, &conv.ParticipantOne, &conv.ParticipantTwo, &conv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

// FindAllByUser returns paginated conversations for a user, including the other
// participant's details and the last message. Ordered by most recent message first.
// Use with CountByUser to calculate pagination metadata.
func (r *ConversationRepository) FindAllByUser(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error) {
	query := `
		SELECT
			c.id,
			CASE WHEN c.participant_one = $1 THEN c.participant_two ELSE c.participant_one END AS other_user_id,
			u.display_name,
			COALESCE(u.phone_number, ''),
			u.role,
			COALESCE(
				(SELECT m.content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1),
				''
			) AS last_message,
			COALESCE(
				(SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id AND m.sender_id != $1 AND m.read_at IS NULL),
				0
			) AS unread_count,
			c.created_at
		FROM conversations c
		JOIN users u ON u.id = CASE WHEN c.participant_one = $1 THEN c.participant_two ELSE c.participant_one END
		WHERE c.participant_one = $1 OR c.participant_two = $1
		ORDER BY
			(SELECT m.created_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1) DESC NULLS LAST
		OFFSET $2 LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []models.ConversationWithDetails
	for rows.Next() {
		var c models.ConversationWithDetails
		if err := rows.Scan(
			&c.ID, &c.OtherUser.ID, &c.OtherUser.DisplayName,
			&c.OtherUser.PhoneNumber, &c.OtherUser.Role,
			&c.LastMessage, &c.UnreadCount, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		conversations = append(conversations, c)
	}

	if conversations == nil {
		conversations = []models.ConversationWithDetails{}
	}

	return conversations, nil
}

// FindAllForAdmin returns paginated customer conversations for the shared
// admin inbox. It always exposes the customer side of the thread so any admin
// can review and reply to the same conversation.
func (r *ConversationRepository) FindAllForAdmin(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error) {
	query := `
		SELECT
			c.id,
			CASE WHEN u1.role = 'customer' THEN u1.id ELSE u2.id END AS customer_user_id,
			CASE WHEN u1.role = 'customer' THEN u1.display_name ELSE u2.display_name END,
			CASE WHEN u1.role = 'customer' THEN COALESCE(u1.phone_number, '') ELSE COALESCE(u2.phone_number, '') END,
			CASE WHEN u1.role = 'customer' THEN u1.role ELSE u2.role END,
			COALESCE(
				(SELECT m.content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1),
				''
			) AS last_message,
			COALESCE(
				(SELECT COUNT(*)
				 FROM messages m
				 JOIN users sender ON sender.id = m.sender_id
				 WHERE m.conversation_id = c.id
				   AND sender.role = 'customer'
				   AND m.read_at IS NULL),
				0
			) AS unread_count,
			c.created_at
		FROM conversations c
		JOIN users u1 ON u1.id = c.participant_one
		JOIN users u2 ON u2.id = c.participant_two
		WHERE u1.role = 'customer' OR u2.role = 'customer'
		ORDER BY
			(SELECT m.created_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1) DESC NULLS LAST
		OFFSET $1 LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query admin conversations: %w", err)
	}
	defer rows.Close()

	var conversations []models.ConversationWithDetails
	for rows.Next() {
		var c models.ConversationWithDetails
		if err := rows.Scan(
			&c.ID, &c.OtherUser.ID, &c.OtherUser.DisplayName,
			&c.OtherUser.PhoneNumber, &c.OtherUser.Role,
			&c.LastMessage, &c.UnreadCount, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin conversation: %w", err)
		}
		conversations = append(conversations, c)
	}

	if conversations == nil {
		conversations = []models.ConversationWithDetails{}
	}

	return conversations, nil
}

// CountByUser returns the total number of conversations for a user.
func (r *ConversationRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM conversations
		WHERE participant_one = $1 OR participant_two = $1
	`

	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count conversations by user: %w", err)
	}

	return count, nil
}

// CountAllForAdmin returns the total number of customer conversations visible
// in the shared admin inbox.
func (r *ConversationRepository) CountAllForAdmin(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM conversations c
		JOIN users u1 ON u1.id = c.participant_one
		JOIN users u2 ON u2.id = c.participant_two
		WHERE u1.role = 'customer' OR u2.role = 'customer'
	`

	var count int
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count admin conversations: %w", err)
	}

	return count, nil
}

// FindByID returns a single conversation by UUID. Used to verify
// ownership before sending messages.
func (r *ConversationRepository) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	query := `
		SELECT id, participant_one, participant_two, created_at
		FROM conversations
		WHERE id = $1
	`

	var conv models.Conversation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.ParticipantOne, &conv.ParticipantTwo, &conv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find conversation by id: %w", err)
	}

	return &conv, nil
}

// sortPair ensures (A, B) and (B, A) produce the same ordered pair.
// This is how we guarantee one conversation per user pair.
func sortPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
