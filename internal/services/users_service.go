package services

import (
    "context"
)

// UsersService handles user profile and status management; excludes password and tokens.
type UsersService interface {
    // GetByID returns a user by ID.
    GetByID(ctx context.Context, id int) (*UserDTO, error)
    // GetByName returns a user by exact name.
    GetByName(ctx context.Context, name string) (*UserDTO, error)
    // List returns users matching filter.
    List(ctx context.Context, filter UsersFilter) ([]UserDTO, error)

    // Create creates a user with role and optional group/note.
    Create(ctx context.Context, name string, role string, group *string, note *string) (*UserDTO, error)
    // UpdateProfile updates non-sensitive profile fields: name/user_group/note.
    UpdateProfile(ctx context.Context, userID int, fields UpdateUserFields) (*UserDTO, error)
    // AssignRole updates user's role; sensitive operation with policy checks.
    AssignRole(ctx context.Context, userID int, role string) error
    // SetActive enables or disables a user; sensitive operation.
    SetActive(ctx context.Context, userID int, active bool) error
    // Delete removes a user.
    Delete(ctx context.Context, userID int) error
}

// UpdateUserFields contains non-sensitive profile fields to update.
type UpdateUserFields struct {
    Name      *string
    UserGroup *string
    Note      *string
}

// UsersFilter contains filters for listing users.
// Usage:
// - Query: case-insensitive fuzzy search against `name` and `note`.
// - Name: exact match; when set, service prefers GetByName then applies remaining conditions in-memory.
// - Role: exact filter by role.
// - Active: tri-state (nil=no filter; true/false=filter by status).
// - UserGroup: exact filter by group.
// Notes:
// - When Name is not set, the service maps filters to repository parameters; DB uses ILIKE and orders by name ASC.
// - No pagination; List returns all matching entries.
type UsersFilter struct {
    // Query performs fuzzy search on name and note.
    Query     *string
    Name      *string
    Role      *string
    Active    *bool
    UserGroup *string
}

// UserDTO represents a safe user view for output.
type UserDTO struct {
    UserID    int
    Name      string
    Role      string
    IsActive  bool
    UserGroup *string
    Note      *string
}