package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"ai-notetaking-be/pkg/embedding"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//	{
//	    "model": "models/gemini-embedding-exp-12-22",
//	    "content": {
//	        "parts": [
//	            {
//	                "text": "What is the meaning of life?"
//	            }
//	        ]
//	    }
//	}

type IConsumerService interface {
	Consume(ctx context.Context) error
}

type consumerService struct {
	notebookRepository      repository.INotebookRepository
	noteRepository          repository.INoteRepository
	noteEmbeddingRepository repository.INoteEmbeddingRepository
	pubSub                  *gochannel.GoChannel
	topicName               string

	db *pgxpool.Pool
}

func (cs *consumerService) Consume(ctx context.Context) error {
	messages, err := cs.pubSub.Subscribe(ctx, cs.topicName)
	if err != nil {
		return err
	}

	go func() {
		for msg := range messages {
			cs.processMessage(ctx, msg)
		}
	}()

	return nil
}

func (cs *consumerService) processMessage(ctx context.Context, msg *message.Message) {
	select {
	case <-ctx.Done():
		log.Warn("context cancelled")
		msg.Nack()
		return
	default:
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error(r)
			msg.Nack()
		}
	}()

	var payload dto.PublishEmbedNotMessage
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		log.Error(err)
		panic(err)
	}
	note, err := cs.noteRepository.GetById(ctx, payload.NoteId)
	if err != nil {
		panic(err)
	}
	notebook, err := cs.notebookRepository.GetById(ctx, note.NotebookId)
	if err != nil {
		panic(err)
	}
	noteUpdatedAt := "-"
	if note.UpdatedAt != nil {
		noteUpdatedAt = note.UpdatedAt.Format(time.RFC3339)
	}
	// content DIPAKAI + fmt.Sprintf
	content := fmt.Sprintf(
		`Note Title: %s
		Notebook Title: %s

		%s

		Created At: %s
		Updated At: %s`,
		note.Title,
		notebook.Name,
		note.Content,
		note.CreatedAt.Format(time.RFC3339),
		noteUpdatedAt,
	)
	// embedding pakai content
	res, err := embedding.GetGeminiEmbedding(
		os.Getenv("GOOGLE_GEMINI_API_KEY"),
		content,
		"RETRIEVAL_DOCUMENT",
	)
	if err != nil {
		panic(err)
	}

	noteEmbedding := entity.NoteEmbedding{
		Id:             uuid.New(),
		Document:       note.Content,
		EmbeddingValue: res.Embedding.Values,
		NoteId:         note.Id,
		CreatedAt:      time.Now(),
	}
	tx, err := cs.db.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)
	noteEmbeddingRepository := cs.noteEmbeddingRepository.UsingTx(ctx, tx)
	err = noteEmbeddingRepository.DeleteByNoteId(ctx, note.Id)
	if err != nil {
		panic(err)
	}
	if err := noteEmbeddingRepository.Create(ctx, &noteEmbedding); err != nil {
		log.Error(err)
		msg.Nack()
		return
	}
	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
	msg.Ack()
}

func NewConsumerService(
	pubSub *gochannel.GoChannel,
	topicName string, noteRepository repository.INoteRepository,
	noteEmbeddingRepository repository.INoteEmbeddingRepository,
	notebookRepository repository.INotebookRepository,
	db *pgxpool.Pool,

) IConsumerService {
	return &consumerService{
		pubSub:                  pubSub,
		topicName:               topicName,
		noteRepository:          noteRepository,
		noteEmbeddingRepository: noteEmbeddingRepository,
		notebookRepository:      notebookRepository,
		db:                      db,
	}
}
