package repositories

import (
    "context"
    "database/sql"
    "strconv"
    "strings"

    "cutrix-backend/internal/models"
)

// SqlUsersRepository implements UsersRepository against PostgreSQL.
type SqlUsersRepository struct{ db *sql.DB }

// NewSqlUsersRepository creates a new SQL-based users repository.
func NewSqlUsersRepository(db *sql.DB) *SqlUsersRepository { return &SqlUsersRepository{db: db} }

// Compile-time check that SqlUsersRepository satisfies UsersRepository.
var _ UsersRepository = (*SqlUsersRepository)(nil)

type scanner interface{ Scan(dest ...any) error }

func scanUser(s scanner) (*models.User, error) {
    var u models.User
    var g, n sql.NullString
    if err := s.Scan(&u.UserID, &u.Name, &u.PasswordHash, &u.Role, &u.IsActive, &g, &n); err != nil { return nil, err }
    if g.Valid { v := g.String; u.Group = &v }
    if n.Valid { v := n.String; u.Note = &v }
    return &u, nil
}

// Create creates a user and returns the generated ID.
func (r *SqlUsersRepository) Create(ctx context.Context, user *models.User) (int, error) {
    const q = `
        INSERT INTO public.users (name, password_hash, role, is_active, user_group, note)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING user_id`
    var id int
    err := r.db.QueryRowContext(ctx, q, user.Name, user.PasswordHash, user.Role, user.IsActive, user.Group, user.Note).Scan(&id)
    if err == nil { user.UserID = id }
    return id, err
}

// Delete removes a user by ID.
func (r *SqlUsersRepository) Delete(ctx context.Context, id int) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM public.users WHERE user_id = $1`, id)
    return err
}

// Update persists all mutable fields of the user.
func (r *SqlUsersRepository) Update(ctx context.Context, user *models.User) error {
    const q = `
        UPDATE public.users
        SET name = $1, password_hash = $2, role = $3, is_active = $4, user_group = $5, note = $6
        WHERE user_id = $7`
    _, err := r.db.ExecContext(ctx, q, user.Name, user.PasswordHash, user.Role, user.IsActive, user.Group, user.Note, user.UserID)
    return err
}

// GetAll returns all users ordered by name.
func (r *SqlUsersRepository) GetAll(ctx context.Context) ([]models.User, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT user_id, name, password_hash, role, is_active, user_group, note
        FROM public.users
        ORDER BY name ASC`)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.User
    for rows.Next() {
        u, err := scanUser(rows)
        if err != nil { return nil, err }
        res = append(res, *u)
    }
    return res, rows.Err()
}

// GetByID returns a user by ID.
func (r *SqlUsersRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT user_id, name, password_hash, role, is_active, user_group, note
        FROM public.users WHERE user_id = $1`, id)
    return scanUser(row)
}

// GetByName returns a user by unique name.
func (r *SqlUsersRepository) GetByName(ctx context.Context, name string) (*models.User, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT user_id, name, password_hash, role, is_active, user_group, note
        FROM public.users WHERE name = $1`, name)
    return scanUser(row)
}

// UpdateName updates the user's name.
func (r *SqlUsersRepository) UpdateName(ctx context.Context, id int, name string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET name = $1 WHERE user_id = $2`, name, id)
    return err
}

// UpdatePasswordHash updates the user's password hash.
func (r *SqlUsersRepository) UpdatePasswordHash(ctx context.Context, id int, passwordHash string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET password_hash = $1 WHERE user_id = $2`, passwordHash, id)
    return err
}

// UpdateRole updates the user's role.
func (r *SqlUsersRepository) UpdateRole(ctx context.Context, id int, role string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET role = $1 WHERE user_id = $2`, role, id)
    return err
}

// SetActive toggles the user's active status.
func (r *SqlUsersRepository) SetActive(ctx context.Context, id int, active bool) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET is_active = $1 WHERE user_id = $2`, active, id)
    return err
}

// UpdateGroup updates the user's group.
func (r *SqlUsersRepository) UpdateGroup(ctx context.Context, id int, group string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET user_group = $1 WHERE user_id = $2`, group, id)
    return err
}

// UpdateNote updates the user's note.
func (r *SqlUsersRepository) UpdateNote(ctx context.Context, id int, note string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE public.users SET note = $1 WHERE user_id = $2`, note, id)
    return err
}

// List returns users filtered by the provided UsersFilter.
func (r *SqlUsersRepository) List(ctx context.Context, role *string, group *string, active *bool, query *string) ([]models.User, error) {
    base := `SELECT user_id, name, password_hash, role, is_active, user_group, note FROM public.users`
    var conds []string
    var args []any
    idx := 1
    if role != nil { conds = append(conds, `role = $`+strconv.Itoa(idx)); args = append(args, *role); idx++ }
    if group != nil { conds = append(conds, `user_group = $`+strconv.Itoa(idx)); args = append(args, *group); idx++ }
    if active != nil { conds = append(conds, `is_active = $`+strconv.Itoa(idx)); args = append(args, *active); idx++ }
    if query != nil { conds = append(conds, `(name ILIKE $`+strconv.Itoa(idx)+` OR note ILIKE $`+strconv.Itoa(idx)+`)`); args = append(args, `%`+*query+`%`); idx++ }
    q := base
    if len(conds) > 0 { q += ` WHERE ` + strings.Join(conds, ` AND `) }
    q += ` ORDER BY name ASC`
    rows, err := r.db.QueryContext(ctx, q, args...)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.User
    for rows.Next() {
        u, err := scanUser(rows)
        if err != nil { return nil, err }
        res = append(res, *u)
    }
    return res, rows.Err()
}

// Count returns the number of users that match the filter.
func (r *SqlUsersRepository) Count(ctx context.Context, role *string, group *string, active *bool, query *string) (int, error) {
    base := `SELECT COUNT(*) FROM public.users`
    var conds []string
    var args []any
    idx := 1
    if role != nil { conds = append(conds, `role = $`+strconv.Itoa(idx)); args = append(args, *role); idx++ }
    if group != nil { conds = append(conds, `user_group = $`+strconv.Itoa(idx)); args = append(args, *group); idx++ }
    if active != nil { conds = append(conds, `is_active = $`+strconv.Itoa(idx)); args = append(args, *active); idx++ }
    if query != nil { conds = append(conds, `(name ILIKE $`+strconv.Itoa(idx)+` OR note ILIKE $`+strconv.Itoa(idx)+`)`); args = append(args, `%`+*query+`%`); idx++ }
    q := base
    if len(conds) > 0 { q += ` WHERE ` + strings.Join(conds, ` AND `) }
    var count int
    err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
    return count, err
}

// ListActive returns all active users.
func (r *SqlUsersRepository) ListActive(ctx context.Context) ([]models.User, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT user_id, name, password_hash, role, is_active, user_group, note
        FROM public.users WHERE is_active = TRUE ORDER BY name ASC`)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.User
    for rows.Next() {
        u, err := scanUser(rows)
        if err != nil { return nil, err }
        res = append(res, *u)
    }
    return res, rows.Err()
}

// ExistsByName reports whether a user with the given name exists.
func (r *SqlUsersRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
    var exists bool
    err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM public.users WHERE name = $1)`, name).Scan(&exists)
    return exists, err
}
