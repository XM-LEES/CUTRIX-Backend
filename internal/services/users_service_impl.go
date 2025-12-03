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
func (s *usersService) Create(ctx context.Context, currentUserID int, currentUserRole string, name string, role string, group *string, note *string) (*UserDTO, error) {
    // Check if name already exists
    exists, err := s.repo.ExistsByName(ctx, name)
    if err != nil { return nil, err }
    if exists { return nil, ErrConflict }

    // Business rule: Manager cannot create Admin
    if currentUserRole == "manager" && role == "admin" {
        return nil, ErrForbidden
    }

    // Business rule: Manager cannot create Manager
    if currentUserRole == "manager" && role == "manager" {
        return nil, ErrForbidden
    }

    // Business rule: System can only have one Admin
    if role == "admin" {
        adminRole := "admin"
        existing, err := s.repo.List(ctx, &adminRole, nil, nil, nil)
        if err != nil { return nil, err }
        // Filter active admins only
        for _, u := range existing {
            if u.Role == "admin" && u.IsActive {
                return nil, ErrForbidden
            }
        }
    }

    // Business rule: System can only have one Manager
    if role == "manager" {
        managerRole := "manager"
        existing, err := s.repo.List(ctx, &managerRole, nil, nil, nil)
        if err != nil { return nil, err }
        // Filter active managers only
        for _, u := range existing {
            if u.Role == "manager" && u.IsActive {
                return nil, ErrForbidden
            }
        }
    }

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
func (s *usersService) AssignRole(ctx context.Context, currentUserID int, currentUserRole string, targetUserID int, role string) error {
    // Get target user to check their current role
    targetUser, err := s.repo.GetByID(ctx, targetUserID)
    if err != nil { return err }
    if targetUser == nil { return ErrNotFound }

    // Business rule: Manager cannot operate on Admin
    if currentUserRole == "manager" && targetUser.Role == "admin" {
        return ErrForbidden
    }

    // Business rule: Admin/Manager cannot modify their own role
    if (currentUserRole == "admin" || currentUserRole == "manager") && currentUserID == targetUserID {
        return ErrForbidden
    }

    // Business rule: System can only have one Admin
    if role == "admin" && targetUser.Role != "admin" {
        adminRole := "admin"
        existing, err := s.repo.List(ctx, &adminRole, nil, nil, nil)
        if err != nil { return err }
        // Check if there's already an active admin
        for _, u := range existing {
            if u.Role == "admin" && u.IsActive && u.UserID != targetUserID {
                return ErrForbidden
            }
        }
    }

    // Business rule: System can only have one Manager
    if role == "manager" && targetUser.Role != "manager" {
        managerRole := "manager"
        existing, err := s.repo.List(ctx, &managerRole, nil, nil, nil)
        if err != nil { return err }
        // Check if there's already an active manager
        for _, u := range existing {
            if u.Role == "manager" && u.IsActive && u.UserID != targetUserID {
                return ErrForbidden
            }
        }
    }

    return s.repo.UpdateRole(ctx, targetUserID, role)
}

// SetActive enables or disables a user; sensitive operation.
func (s *usersService) SetActive(ctx context.Context, currentUserID int, currentUserRole string, targetUserID int, active bool) error {
    // Get target user to check their role
    targetUser, err := s.repo.GetByID(ctx, targetUserID)
    if err != nil { return err }
    if targetUser == nil { return ErrNotFound }

    // Business rule: Manager cannot operate on Admin
    if currentUserRole == "manager" && targetUser.Role == "admin" {
        return ErrForbidden
    }

    // Business rule: Admin/Manager cannot deactivate themselves
    if (currentUserRole == "admin" || currentUserRole == "manager") && currentUserID == targetUserID && !active {
        return ErrForbidden
    }

    return s.repo.SetActive(ctx, targetUserID, active)
}

// Delete removes a user.
func (s *usersService) Delete(ctx context.Context, currentUserID int, currentUserRole string, targetUserID int) error {
    // Get target user to check their role
    targetUser, err := s.repo.GetByID(ctx, targetUserID)
    if err != nil { return err }
    if targetUser == nil { return ErrNotFound }

    // Business rule: Manager cannot delete Admin
    if currentUserRole == "manager" && targetUser.Role == "admin" {
        return ErrForbidden
    }

    // Business rule: Admin/Manager cannot delete themselves
    if (currentUserRole == "admin" || currentUserRole == "manager") && currentUserID == targetUserID {
        return ErrForbidden
    }

    return s.repo.Delete(ctx, targetUserID)
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