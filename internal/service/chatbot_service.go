package service

import (
	"ai-notetaking-be/internal/constant"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IChatbotService interface {
	CreateSession(ctx context.Context) (*dto.CreateSessionResponse, error)
	GetAllSesions(ctx context.Context) ([]*dto.GetAllSessionsResponse, error)
	GetChatHistory(ctx context.Context, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error)
	SendChat(ctx context.Context, request *dto.SendChatRequest) (*dto.SendChatResponse, error)
}

type chatbotService struct {
	db                       *pgxpool.Pool
	chatSessionRepository    repository.IChatSessionRepository
	chatMessageRepository    repository.IChatMessageRepository
	chatMessageRawRepository repository.IChatMessageRawRepository
}

func (cs *chatbotService) CreateSession(ctx context.Context) (*dto.CreateSessionResponse, error) {
	now := time.Now()

	chatSession := entity.ChatSession{
		Id:        uuid.New(),
		Title:     "Unnamed session",
		CreatedAt: now,
	}

	chatMessage := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          "Hi, how can I help you?",
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}

	chatMessageRawUser := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialUserPromptv1,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now,
	}

	chatMessageRawModel := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          constant.ChatMessageRawInitialModelPromptv1,
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: chatSession.Id,
		CreatedAt:     now.Add(1 * time.Second),
	}

	tx, err := cs.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	chatSessionRepo := cs.chatSessionRepository.UsingTx(ctx, tx)
	chatMessageRepo := cs.chatMessageRepository.UsingTx(ctx, tx)
	chatMessageRawRepo := cs.chatMessageRawRepository.UsingTx(ctx, tx)

	if err := chatSessionRepo.Create(ctx, &chatSession); err != nil {
		return nil, err
	}
	if err := chatMessageRepo.Create(ctx, &chatMessage); err != nil {
		return nil, err
	}
	if err := chatMessageRawRepo.Create(ctx, &chatMessageRawUser); err != nil {
		return nil, err
	}
	if err := chatMessageRawRepo.Create(ctx, &chatMessageRawModel); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &dto.CreateSessionResponse{
		Id: chatSession.Id,
	}, nil
}

func (cs *chatbotService) GetAllSesions(ctx context.Context) ([]*dto.GetAllSessionsResponse, error) {
	chatSession, err := cs.chatSessionRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.GetAllSessionsResponse, 0)
	for _, chatchatSession := range chatSession {
		response = append(response, &dto.GetAllSessionsResponse{
			Id:        chatchatSession.Id,
			Title:     chatchatSession.Title,
			CreatedAt: chatchatSession.CreatedAt,
			UpdatedAt: chatchatSession.UpdatedAt,
		})
	}
	return response, nil
}

func (cs *chatbotService) GetChatHistory(ctx context.Context, sessionId uuid.UUID) ([]*dto.GetChatHistoryResponse, error) {
	_, err := cs.chatSessionRepository.GetById(ctx, sessionId)
	if err != nil {
		return nil, err
	}
	chatMessage, err := cs.chatMessageRepository.GetByChatSessionId(ctx, sessionId)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.GetChatHistoryResponse, 0)
	for _, chatMessage := range chatMessage {
		response = append(response, &dto.GetChatHistoryResponse{
			Id:        chatMessage.Id,
			Role:      chatMessage.Role,
			Chat:      chatMessage.Chat,
			CreatedAt: chatMessage.CreatedAt,
		})
	}
	return response, nil
}

func (cs *chatbotService) SendChat(ctx context.Context, request *dto.SendChatRequest) (*dto.SendChatResponse, error) {
	tx, err := cs.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	chatSessionRepository := cs.chatSessionRepository.UsingTx(ctx, tx)
	chatMessageRepository := cs.chatMessageRepository.UsingTx(ctx, tx)
	chatMessageRawRepository := cs.chatMessageRawRepository.UsingTx(ctx, tx)

	chatSession, err := chatSessionRepository.GetById(ctx, request.ChatSessionId)
	if err != nil {
		return nil, err
	}

	existingRawChats, err := chatMessageRawRepository.GetByChatSessionId(ctx, request.ChatSessionId)
	if err != nil {
		return nil, err
	}
	updateSessionTitle := len(existingRawChats) == 2

	now := time.Now()
	chatMessage := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          request.Chat,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now,
	}
	chatMessageRaw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          request.Chat,
		Role:          constant.ChatMessageRoleUser,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now,
	}
	chatMessageModel := entity.ChatMessage{
		Id:            uuid.New(),
		Chat:          "This is automated dumy response",
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now.Add(1 * time.Millisecond),
	}
	chatMessageModelRaw := entity.ChatMessageRaw{
		Id:            uuid.New(),
		Chat:          "This is automated dumy response",
		Role:          constant.ChatMessageRoleModel,
		ChatSessionId: request.ChatSessionId,
		CreatedAt:     now.Add(1 * time.Millisecond),
	}

	chatMessageRepository.Create(ctx, &chatMessage)
	chatMessageRepository.Create(ctx, &chatMessageModel)
	chatMessageRawRepository.Create(ctx, &chatMessageRaw)
	chatMessageRawRepository.Create(ctx, &chatMessageModelRaw)

	if updateSessionTitle {
		chatSession.Title = request.Chat
		chatSession.UpdatedAt = &now
		err = chatSessionRepository.Update(ctx, chatSession)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.SendChatResponse{
		ChatSessionId:    chatSession.Id,
		ChatSessionTitle: chatSession.Title,
		Sent: &dto.SendChatResponseChat{
			Id:        chatMessage.Id,
			Chat:      chatMessage.Chat,
			Role:      chatMessage.Role,
			CreatedAt: chatMessage.CreatedAt,
		},
		Reply: &dto.SendChatResponseChat{
			Id:        chatMessageModel.Id,
			Chat:      chatMessageModel.Chat,
			Role:      chatMessageModel.Role,
			CreatedAt: chatMessageModel.CreatedAt,
		},
	}, nil
}
func NewChatbotService(
	db *pgxpool.Pool,
	chatSessionRepository repository.IChatSessionRepository,
	chatMessageRepository repository.IChatMessageRepository,
	chatMessageRawRepository repository.IChatMessageRawRepository,
) IChatbotService {
	return &chatbotService{
		db:                       db,
		chatSessionRepository:    chatSessionRepository,
		chatMessageRepository:    chatMessageRepository,
		chatMessageRawRepository: chatMessageRawRepository,
	}
}
