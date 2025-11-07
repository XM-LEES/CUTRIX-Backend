package services

import (
    "context"
    "strings"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

// usersService implements UsersService using UsersRepository.
type usersService struct {
    repo repositories.UsersRepository
}

// NewUsersService constructs a UsersService.
func NewUsersService(repo repositories.UsersRepository) UsersService {
    return &usersService{repo: repo}
}

// GetByID returns a user by ID.
func (s *usersService) GetByID(ctx context.Context, id int) (*UserDTO, error) {
    u, err := s.repo.GetByID(ctx, id)
    if err != nil || u == nil { return nil, err }
    return toDTO(u), nil
}

// GetByName returns a user by exact name.
func (s *usersService) GetByName(ctx context.Context, name string) (*UserDTO, error) {
    u, err := s.repo.GetByName(ctx, name)
    if err != nil || u == nil { return nil, err }
    return toDTO(u), nil
}

// List returns users matching filter.
func (s *usersService) List(ctx context.Context, filter UsersFilter) ([]UserDTO, error) {
    // If exact name specified, prefer direct lookup to avoid fuzzy matches.
    if filter.Name != nil && *filter.Name != "" {
        u, err := s.repo.GetByName(ctx, *filter.Name)
        if err != nil { return nil, err }
        if u == nil { return []UserDTO{}, nil }
        // Apply other filters locally if provided.
        if filter.Role != nil && u.Role != *filter.Role { return []UserDTO{}, nil }
        if filter.Active != nil && u.IsActive != *filter.Active { return []UserDTO{}, nil }
        if filter.UserGroup != nil {
            if u.Group == nil || *u.Group != *filter.UserGroup { return []UserDTO{}, nil }
        }
        if filter.Query != nil {
            q := *filter.Query
            match := containsFold(u.Name, q) || (u.Note != nil && containsFold(*u.Note, q))
            if !match { return []UserDTO{}, nil }
        }
        return []UserDTO{*toDTO(u)}, nil
    }

    users, err := s.repo.List(ctx, filter.Role, filter.UserGroup, filter.Active, filter.Query)
    if err != nil { return nil, err }
    out := make([]UserDTO, 0, len(users))
    for i := range users { out = append(out, *toDTO(&users[i])) }
    return out, nil
}

// Create creates a user with role and optional group/note.
func (s *usersService) Create(ctx context.Context, name string, role string, group *string, note *string) (*UserDTO, error) {
    exists, err := s.repo.ExistsByName(ctx, name)
    if err != nil { return nil, err }
    if exists { return nil, ErrConflict }

    u := &models.User{
        Name:         name,
        PasswordHash: "", // password managed by AuthService
        Role:         role,
        IsActive:     true,
        Group:        group,
        Note:         note,
    }
    _, err = s.repo.Create(ctx, u)
    if err != nil { return nil, err }
    return toDTO(u), nil
}

// UpdateProfile updates non-sensitive profile fields: name/user_group/note.
func (s *usersService) UpdateProfile(ctx context.Context, userID int, fields UpdateUserFields) (*UserDTO, error) {
    if fields.Name != nil {
        curr, err := s.repo.GetByID(ctx, userID)
        if err != nil { return nil, err }
        if curr == nil { return nil, ErrNotFound }
        if curr.Name != *fields.Name {
            exists, err := s.repo.ExistsByName(ctx, *fields.Name)
            if err != nil { return nil, err }
            if exists { return nil, ErrConflict }
            if err := s.repo.UpdateName(ctx, userID, *fields.Name); err != nil { return nil, err }
        }
    }
    if fields.UserGroup != nil {
        if err := s.repo.UpdateGroup(ctx, userID, *fields.UserGroup); err != nil { return nil, err }
    }
    if fields.Note != nil {
        if err := s.repo.UpdateNote(ctx, userID, *fields.Note); err != nil { return nil, err }
    }
    u, err := s.repo.GetByID(ctx, userID)
    if err != nil || u == nil { if err == nil { err = ErrNotFound }; return nil, err }
    return toDTO(u), nil
}

// AssignRole updates user's role; sensitive operation with policy checks.
func (s *usersService) AssignRole(ctx context.Context, userID int, role string) error {
    return s.repo.UpdateRole(ctx, userID, role)
}

// SetActive enables or disables a user; sensitive operation.
func (s *usersService) SetActive(ctx context.Context, userID int, active bool) error {
    return s.repo.SetActive(ctx, userID, active)
}

// Delete removes a user.
func (s *usersService) Delete(ctx context.Context, userID int) error {
    return s.repo.Delete(ctx, userID)
}

// toDTO converts models.User to UserDTO.
func toDTO(u *models.User) *UserDTO {
    return &UserDTO{
        UserID:    u.UserID,
        Name:      u.Name,
        Role:      u.Role,
        IsActive:  u.IsActive,
        UserGroup: u.Group,
        Note:      u.Note,
    }
}

// containsFold reports whether s contains sub, case-insensitive.
func containsFold(s, sub string) bool {
    return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}