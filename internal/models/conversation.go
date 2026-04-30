package models

import "time"

// Conversation is a one-on-one chat thread between two users. For now,
// participant_2 is always the admin, but the schema supports any two users.
// When we open up customer-to-customer chat, no migration is needed.
type Conversation struct {
	ID             string    `json:"id"`
	ParticipantOne string    `json:"participant_one"`
	ParticipantTwo string    `json:"participant_two"`
	CreatedAt      time.Time `json:"created_at"`
}

// ConversationUser is the lightweight profile attached to conversation list
// responses. It intentionally excludes timestamps and account internals.
type ConversationUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"`
}

// ConversationWithDetails is the shape returned to the frontend. It includes
// the other user's profile and the most recent message so the conversation
// list view doesn't need N+1 queries.
type ConversationWithDetails struct {
	ID          string           `json:"id"`
	OtherUser   ConversationUser `json:"other_user"`
	LastMessage *string          `json:"last_message,omitempty"`
	UnreadCount int              `json:"unread_count"`
	CreatedAt   time.Time        `json:"created_at"`
}
