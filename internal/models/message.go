package models

import "time"

// Message is a single chat message within a conversation. ReadAt is nil
// until the recipient views the message. SenderID determines bubble
// alignment on the frontend, left for their messages, right for mine.
type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	SenderID       string     `json:"sender_id"`
	Content        string     `json:"content"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
