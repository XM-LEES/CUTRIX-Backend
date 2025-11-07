package services

import "errors"

// Unified error variables used across services and handlers.
var (
    ErrUnauthorized = errors.New("unauthorized")
    ErrForbidden    = errors.New("forbidden")
    ErrConflict     = errors.New("conflict")
    ErrNotFound     = errors.New("not found")
    ErrValidation   = errors.New("validation error")
)