package services

import (
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

type LayoutsServiceImpl struct { repo repositories.LayoutsRepository }

func NewLayoutsService(repo repositories.LayoutsRepository) *LayoutsServiceImpl { return &LayoutsServiceImpl{repo: repo} }

func (s *LayoutsServiceImpl) Create(layout *models.CuttingLayout) error {
    if layout == nil { return errors.New("nil layout") }
    if layout.PlanID == 0 { return errors.New("plan_id required") }
    if layout.LayoutName == "" { return errors.New("layout_name required") }
    return s.repo.Create(layout)
}

func (s *LayoutsServiceImpl) Delete(id int) error {
    if id <= 0 { return errors.New("invalid layout_id") }
    return s.repo.Delete(id)
}