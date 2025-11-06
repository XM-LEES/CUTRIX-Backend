package services

import (
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

type TasksServiceImpl struct { repo repositories.TasksRepository }

func NewTasksService(repo repositories.TasksRepository) *TasksServiceImpl { return &TasksServiceImpl{repo: repo} }

func (s *TasksServiceImpl) Create(task *models.ProductionTask) error {
    if task == nil { return errors.New("nil task") }
    if task.LayoutID == 0 { return errors.New("layout_id required") }
    if task.Color == "" { return errors.New("color required") }
    if task.PlannedLayers < 0 { return errors.New("planned_layers must be >= 0") }
    return s.repo.Create(task)
}

func (s *TasksServiceImpl) GetByID(id int) (*models.ProductionTask, error) {
    if id <= 0 { return nil, errors.New("invalid task_id") }
    return s.repo.GetByID(id)
}