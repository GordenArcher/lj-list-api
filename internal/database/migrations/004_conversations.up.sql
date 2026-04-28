CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    participant_one UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_two UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_participants UNIQUE (participant_one, participant_two),
    CONSTRAINT different_participants CHECK (participant_one != participant_two)
);

CREATE INDEX idx_conversations_participant_one ON conversations (participant_one);
CREATE INDEX idx_conversations_participant_two ON conversations (participant_two);
