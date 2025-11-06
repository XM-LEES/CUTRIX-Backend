package repositories

import (
    "database/sql"
    "errors"

    "example-demo/internal/models"
)

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) InitSchema() error {
    _, err := r.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`)
    return err
}

func (r *UserRepository) Create(name, email string) (*models.User, error) {
    res, err := r.db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
    if err != nil {
        return nil, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, err
    }
    return r.Get(id)
}

func (r *UserRepository) List() ([]models.User, error) {
    rows, err := r.db.Query("SELECT id, name, email, created_at FROM users ORDER BY id DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []models.User
    for rows.Next() {
        var u models.User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
            return nil, err
        }
        out = append(out, u)
    }
    return out, rows.Err()
}

func (r *UserRepository) Get(id int64) (*models.User, error) {
    var u models.User
    err := r.db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", id).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &u, nil
}

func (r *UserRepository) Update(id int64, name *string, email *string) (*models.User, error) {
    curr, err := r.Get(id)
    if err != nil {
        return nil, err
    }
    if curr == nil {
        return nil, nil
    }
    if name != nil {
        curr.Name = *name
    }
    if email != nil {
        curr.Email = *email
    }
    _, err = r.db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", curr.Name, curr.Email, id)
    if err != nil {
        return nil, err
    }
    return r.Get(id)
}

func (r *UserRepository) Delete(id int64) (bool, error) {
    res, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}