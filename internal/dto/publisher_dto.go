package dto

import "github.com/google/uuid"

type PublishEmbedNotMessage struct {
	NoteId uuid.UUID `json:"note_id"`
}
