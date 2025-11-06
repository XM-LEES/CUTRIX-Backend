package services

import (
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

type PlansServiceImpl struct { repo repositories.PlansRepository }

func NewPlansService(repo repositories.PlansRepository) *PlansServiceImpl { return &PlansServiceImpl{repo: repo} }

func (s *PlansServiceImpl) Create(plan *models.ProductionPlan) error {
    if plan == nil { return errors.New("nil plan") }
    if plan.OrderID == 0 { return errors.New("order_id required") }
    if plan.PlanName == "" { plan.PlanName = "Plan" }
    return s.repo.Create(plan)
}

func (s *PlansServiceImpl) Delete(id int) error {
    if id <= 0 { return errors.New("invalid plan_id") }
    return s.repo.Delete(id)
}