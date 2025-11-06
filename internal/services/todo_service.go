package services

import (
    "example-demo/internal/models"
    "example-demo/internal/repositories"
)

type TodoService struct {
    repo *repositories.TodoRepository
}

func NewTodoService(repo *repositories.TodoRepository) *TodoService { return &TodoService{repo: repo} }

func (s *TodoService) Create(title string, userID *int64) (*models.Todo, error) {
    return s.repo.Create(title, userID)
}

func (s *TodoService) List(userID *int64) ([]models.Todo, error) {
    return s.repo.List(userID)
}

func (s *TodoService) Get(id int64) (*models.Todo, error) {
    return s.repo.Get(id)
}

func (s *TodoService) Update(id int64, title *string, completed *bool, userID *int64) (*models.Todo, error) {
    return s.repo.Update(id, title, completed, userID)
}

func (s *TodoService) Delete(id int64) (bool, error) {
    return s.repo.Delete(id)
}