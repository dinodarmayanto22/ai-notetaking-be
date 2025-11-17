package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type INotebookService interface {
	GetAll(ctx context.Context) ([]*dto.GetAllNotebookResponse, error)
	Create(ctx context.Context, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error)
	Show(ctx context.Context, id uuid.UUID) (*dto.ShowNotebookResponse, error)
	Update(ctx context.Context, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MoveNotebook(ctx context.Context, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error)
}

type notebookService struct {
	notebookRepository repository.INotebookRepository
	db                 *pgxpool.Pool
}

func NewNotebookService(notebookRepository repository.INotebookRepository, db *pgxpool.Pool) INotebookService {
	return &notebookService{
		notebookRepository: notebookRepository,
		db:                 db,
	}
}
func (c *notebookService) GetAll(ctx context.Context) ([]*dto.GetAllNotebookResponse, error) {

	notebooks, err := c.notebookRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.GetAllNotebookResponse, 0)
	for _, notebook := range notebooks {
		res := dto.GetAllNotebookResponse{
			Id:        notebook.Id,
			Name:      notebook.Name,
			ParentId:  notebook.ParentId,
			CreatedAt: notebook.CreatedAt,
			UpdatedAt: notebook.UpdatedAt,
		}
		result = append(result, &res)
	}

	return result, nil
}

func (c *notebookService) Create(ctx context.Context, req *dto.CreateNotebookRequest) (*dto.CreateNotebookResponse, error) {

	notebook := entity.NoteBook{
		Id:        uuid.New(),
		Name:      req.Name,
		ParentId:  req.ParentId, // pointer -> pointer ✔
		CreatedAt: time.Now(),   // field exists ✔
	}

	err := c.notebookRepository.Create(ctx, &notebook)
	if err != nil {
		return nil, err
	}

	return &dto.CreateNotebookResponse{
		Id: notebook.Id,
	}, nil
}

func (c *notebookService) Show(ctx context.Context, id uuid.UUID) (*dto.ShowNotebookResponse, error) {
	notebook, err := c.notebookRepository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := dto.ShowNotebookResponse{
		Id:        notebook.Id,
		Name:      notebook.Name,
		ParentId:  notebook.ParentId,
		CreatedAt: notebook.CreatedAt,
		UpdatedAt: notebook.UpdatedAt,
	}

	return &res, nil
}

func (c *notebookService) Update(ctx context.Context, req *dto.UpdateNotebookRequest) (*dto.UpdateNotebookResponse, error) {
	notebook, err := c.notebookRepository.GetById(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	notebook.Name = req.Name
	notebook.UpdatedAt = &now

	err = c.notebookRepository.Update(ctx, notebook)
	if err != nil {
		return nil, err
	}
	res := dto.UpdateNotebookResponse{
		Id: notebook.Id,
	}

	return &res, nil
}

func (c *notebookService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := c.notebookRepository.GetById(ctx, id)
	if err != nil {
		return err
	}

	tx, err := c.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	notebookRepo := c.notebookRepository.UsingTx(ctx, tx)

	err = notebookRepo.DeleteById(ctx, id)
	if err != nil {
		return err
	}

	err = notebookRepo.NullIfParentById(ctx, id)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *notebookService) MoveNotebook(ctx context.Context, req *dto.MoveNotebookRequest) (*dto.MoveNotebookResponse, error) {
	_, err := c.notebookRepository.GetById(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.ParentId != nil {
		_, err = c.notebookRepository.GetById(ctx, *req.ParentId)
		if err != nil {
			return nil, err
		}
	}
	err = c.notebookRepository.UpdateParentId(ctx, req.Id, req.ParentId)
	if err != nil {
		return nil, err
	}

	return &dto.MoveNotebookResponse{
		Id: req.Id,
	}, nil
}
