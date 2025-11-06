package repositories

import (
    "database/sql"
    "errors"

    "example-demo/internal/models"
)

type TodoRepository struct {
    db        *sql.DB
    hasUserID bool
}

func NewTodoRepository(db *sql.DB) *TodoRepository { return &TodoRepository{db: db} }

// InitSchema 创建表（若不存在），并尝试增加 user_id 列用于外键关联
func (r *TodoRepository) InitSchema() error {
    if _, err := r.db.Exec(`
CREATE TABLE IF NOT EXISTS todos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`); err != nil {
        return err
    }
    // 检查是否存在 user_id 列
    var name string
    rows, err := r.db.Query("PRAGMA table_info(todos)")
    if err != nil {
        return err
    }
    defer rows.Close()
    for rows.Next() {
        var cid int
        var ctype, notnull, dfltValue, pk interface{}
        if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
            return err
        }
        if name == "user_id" {
            r.hasUserID = true
            break
        }
    }
    if !r.hasUserID {
        // 尝试添加列；若失败（旧SQLite不支持或列已存在），忽略错误
        if _, err := r.db.Exec("ALTER TABLE todos ADD COLUMN user_id INTEGER"); err == nil {
            r.hasUserID = true
            // 外键约束建议在创建时声明，这里仅示例，运行时依赖 PRAGMA foreign_keys=ON
            // 无法直接为现有列添加外键约束，保持示例简单
        }
    }
    return nil
}

func (r *TodoRepository) Create(title string, userID *int64) (*models.Todo, error) {
    var res sql.Result
    var err error
    if r.hasUserID && userID != nil {
        res, err = r.db.Exec("INSERT INTO todos (title, completed, user_id) VALUES (?, ?, ?)", title, 0, *userID)
    } else {
        res, err = r.db.Exec("INSERT INTO todos (title, completed) VALUES (?, ?)", title, 0)
    }
    if err != nil {
        return nil, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return nil, err
    }
    return r.Get(id)
}

func (r *TodoRepository) List(userID *int64) ([]models.Todo, error) {
    var rows *sql.Rows
    var err error
    if r.hasUserID && userID != nil {
        rows, err = r.db.Query("SELECT id, title, completed, created_at, user_id FROM todos WHERE user_id = ? ORDER BY id DESC", *userID)
    } else if r.hasUserID {
        rows, err = r.db.Query("SELECT id, title, completed, created_at, user_id FROM todos ORDER BY id DESC")
    } else {
        rows, err = r.db.Query("SELECT id, title, completed, created_at FROM todos ORDER BY id DESC")
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []models.Todo
    for rows.Next() {
        var t models.Todo
        var completedInt int
        if r.hasUserID {
            var userIDVal sql.NullInt64
            if err := rows.Scan(&t.ID, &t.Title, &completedInt, &t.CreatedAt, &userIDVal); err != nil {
                return nil, err
            }
            if userIDVal.Valid {
                v := userIDVal.Int64
                t.UserID = &v
            }
        } else {
            if err := rows.Scan(&t.ID, &t.Title, &completedInt, &t.CreatedAt); err != nil {
                return nil, err
            }
        }
        t.Completed = completedInt == 1
        out = append(out, t)
    }
    return out, rows.Err()
}

func (r *TodoRepository) Get(id int64) (*models.Todo, error) {
    var t models.Todo
    var completedInt int
    if r.hasUserID {
        var userIDVal sql.NullInt64
        err := r.db.QueryRow("SELECT id, title, completed, created_at, user_id FROM todos WHERE id = ?", id).Scan(&t.ID, &t.Title, &completedInt, &t.CreatedAt, &userIDVal)
        if err != nil {
            if errors.Is(err, sql.ErrNoRows) {
                return nil, nil
            }
            return nil, err
        }
        if userIDVal.Valid {
            v := userIDVal.Int64
            t.UserID = &v
        }
    } else {
        err := r.db.QueryRow("SELECT id, title, completed, created_at FROM todos WHERE id = ?", id).Scan(&t.ID, &t.Title, &completedInt, &t.CreatedAt)
        if err != nil {
            if errors.Is(err, sql.ErrNoRows) {
                return nil, nil
            }
            return nil, err
        }
    }
    t.Completed = completedInt == 1
    return &t, nil
}

func (r *TodoRepository) Update(id int64, title *string, completed *bool, userID *int64) (*models.Todo, error) {
    curr, err := r.Get(id)
    if err != nil {
        return nil, err
    }
    if curr == nil {
        return nil, nil
    }
    if title != nil {
        curr.Title = *title
    }
    if completed != nil {
        curr.Completed = *completed
    }
    if r.hasUserID {
        curr.UserID = userID
        _, err = r.db.Exec("UPDATE todos SET title = ?, completed = ?, user_id = ? WHERE id = ?", curr.Title, boolToInt(curr.Completed), nullableInt(curr.UserID), id)
    } else {
        _, err = r.db.Exec("UPDATE todos SET title = ?, completed = ? WHERE id = ?", curr.Title, boolToInt(curr.Completed), id)
    }
    if err != nil {
        return nil, err
    }
    return r.Get(id)
}

func (r *TodoRepository) Delete(id int64) (bool, error) {
    res, err := r.db.Exec("DELETE FROM todos WHERE id = ?", id)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func boolToInt(b bool) int {
    if b {
        return 1
    }
    return 0
}

func nullableInt(p *int64) interface{} {
    if p == nil {
        return nil
    }
    return *p
}