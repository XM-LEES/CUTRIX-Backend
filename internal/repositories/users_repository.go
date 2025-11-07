package repositories

import (
    "context"
    "cutrix-backend/internal/models"
)

// UsersRepository defines the user data access contract.
// Notes:
// - To avoid duplicate filter definitions, only the service layer owns UsersFilter.
// - The repository layer exposes parameterized List/Count methods to avoid cyclic dependencies.
type UsersRepository interface {
    // Basic CRUD operations
    Create(ctx context.Context, user *models.User) (int, error)
    Delete(ctx context.Context, id int) error
    Update(ctx context.Context, user *models.User) error
    GetAll(ctx context.Context) ([]models.User, error)
    GetByID(ctx context.Context, id int) (*models.User, error)
    GetByName(ctx context.Context, name string) (*models.User, error)
    // Business operations
    UpdateName(ctx context.Context, id int, name string) error
    UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error
    UpdateRole(ctx context.Context, id int, role string) error
    SetActive(ctx context.Context, id int, active bool) error
    UpdateGroup(ctx context.Context, id int, group string) error
    UpdateNote(ctx context.Context, id int, note string) error

    // List and search operations (parameterized, no repository-specific filter type)
    List(ctx context.Context, role *string, group *string, active *bool, query *string) ([]models.User, error)
    Count(ctx context.Context, role *string, group *string, active *bool, query *string) (int, error)
    ListActive(ctx context.Context) ([]models.User, error)
    ExistsByName(ctx context.Context, name string) (bool, error)
}