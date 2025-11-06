package services

import (
    "example-demo/internal/models"
    "example-demo/internal/repositories"
)

type UserService struct {
    repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService { return &UserService{repo: repo} }

func (s *UserService) Create(name, email string) (*models.User, error) {
    return s.repo.Create(name, email)
}

func (s *UserService) List() ([]models.User, error) {
    return s.repo.List()
}

func (s *UserService) Get(id int64) (*models.User, error) {
    return s.repo.Get(id)
}

func (s *UserService) Update(id int64, name *string, email *string) (*models.User, error) {
    return s.repo.Update(id, name, email)
}

func (s *UserService) Delete(id int64) (bool, error) {
    return s.repo.Delete(id)
}