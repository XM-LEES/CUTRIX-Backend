package repositories

import (
    "context"
    "database/sql"
    "fmt"

    "cutrix-backend/internal/models"
)

type SqlLayoutsRepository struct{ db *sql.DB }

var _ LayoutsRepository = (*SqlLayoutsRepository)(nil)

func NewSqlLayoutsRepository(db *sql.DB) *SqlLayoutsRepository { return &SqlLayoutsRepository{db: db} }

// planStatusByLayout returns the parent plan status for a given layout.
func (r *SqlLayoutsRepository) planStatusByLayout(ctx context.Context, layoutID int) (string, error) {
    const q = `
        SELECT p.status
        FROM production.cutting_layouts l
        JOIN production.plans p ON p.plan_id = l.plan_id
        WHERE l.layout_id = $1`
    var status string
    if err := r.db.QueryRowContext(ctx, q, layoutID).Scan(&status); err != nil { return "", err }
    return status, nil
}

// planStatusByPlan returns the plan status for a given plan.
func (r *SqlLayoutsRepository) planStatusByPlan(ctx context.Context, planID int) (string, error) {
    const q = `SELECT status FROM production.plans WHERE plan_id = $1`
    var status string
    if err := r.db.QueryRowContext(ctx, q, planID).Scan(&status); err != nil { return "", err }
    return status, nil
}

func (r *SqlLayoutsRepository) Create(ctx context.Context, layout *models.CuttingLayout) (int, error) {
    // Pre-check: only allow creating layouts when plan is pending
    status, err := r.planStatusByPlan(ctx, layout.PlanID)
    if err != nil { return 0, err }
    if status != "pending" {
        return 0, fmt.Errorf("计划发布后不允许新增布局 (plan_id=%d, status=%s)", layout.PlanID, status)
    }

    const q = `
        INSERT INTO production.cutting_layouts (plan_id, layout_name, note)
        VALUES ($1, $2, $3)
        RETURNING layout_id`
    var id int
    var note any
    if layout.Note != nil { note = *layout.Note } else { note = nil }
    err = r.db.QueryRowContext(ctx, q, layout.PlanID, layout.LayoutName, note).Scan(&id)
    if err == nil { layout.LayoutID = id }
    return id, err
}

func (r *SqlLayoutsRepository) Delete(ctx context.Context, id int) error {
    // Pre-check: only allow deleting layouts when plan is pending
    status, err := r.planStatusByLayout(ctx, id)
    if err != nil { return err }
    if status != "pending" {
        return fmt.Errorf("计划发布后不允许删除布局 (layout_id=%d, status=%s)", id, status)
    }

    res, err := r.db.ExecContext(ctx, `DELETE FROM production.cutting_layouts WHERE layout_id = $1`, id)
    if err != nil { return err }
    if n, _ := res.RowsAffected(); n == 0 { return sql.ErrNoRows }
    return nil
}

func (r *SqlLayoutsRepository) UpdateName(ctx context.Context, id int, name string) error {
    // Pre-check: only allow updating layout fields when plan is pending
    status, err := r.planStatusByLayout(ctx, id)
    if err != nil { return err }
    if status != "pending" {
        return fmt.Errorf("计划发布后不允许更新布局名称 (layout_id=%d, status=%s)", id, status)
    }
    _, err = r.db.ExecContext(ctx, `UPDATE production.cutting_layouts SET layout_name = $1 WHERE layout_id = $2`, name, id)
    return err
}

func (r *SqlLayoutsRepository) UpdateNote(ctx context.Context, id int, note *string) error {
    // Pre-check: only allow updating layout fields when plan is pending
    status, err := r.planStatusByLayout(ctx, id)
    if err != nil { return err }
    if status != "pending" {
        return fmt.Errorf("计划发布后不允许更新布局备注 (layout_id=%d, status=%s)", id, status)
    }
    _, err = r.db.ExecContext(ctx, `UPDATE production.cutting_layouts SET note = $1 WHERE layout_id = $2`, note, id)
    return err
}

func (r *SqlLayoutsRepository) GetByID(ctx context.Context, id int) (*models.CuttingLayout, error) {
    const q = `SELECT layout_id, plan_id, layout_name, note FROM production.cutting_layouts WHERE layout_id = $1`
    row := r.db.QueryRowContext(ctx, q, id)
    var l models.CuttingLayout
    var note sql.NullString
    if err := row.Scan(&l.LayoutID, &l.PlanID, &l.LayoutName, &note); err != nil { return nil, err }
    if note.Valid { v := note.String; l.Note = &v }
    return &l, nil
}

func (r *SqlLayoutsRepository) List(ctx context.Context) ([]models.CuttingLayout, error) {
    const q = `SELECT layout_id, plan_id, layout_name, note FROM production.cutting_layouts ORDER BY layout_id ASC`
    rows, err := r.db.QueryContext(ctx, q)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.CuttingLayout
    for rows.Next() {
        var l models.CuttingLayout
        var note sql.NullString
        if err := rows.Scan(&l.LayoutID, &l.PlanID, &l.LayoutName, &note); err != nil { return nil, err }
        if note.Valid { v := note.String; l.Note = &v }
        res = append(res, l)
    }
    return res, rows.Err()
}

func (r *SqlLayoutsRepository) ListByPlan(ctx context.Context, planID int) ([]models.CuttingLayout, error) {
    const q = `SELECT layout_id, plan_id, layout_name, note FROM production.cutting_layouts WHERE plan_id = $1 ORDER BY layout_id ASC`
    rows, err := r.db.QueryContext(ctx, q, planID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.CuttingLayout
    for rows.Next() {
        var l models.CuttingLayout
        var note sql.NullString
        if err := rows.Scan(&l.LayoutID, &l.PlanID, &l.LayoutName, &note); err != nil { return nil, err }
        if note.Valid { v := note.String; l.Note = &v }
        res = append(res, l)
    }
    return res, rows.Err()
}

// SetRatios sets size ratios for a layout. Replaces all existing ratios.
func (r *SqlLayoutsRepository) SetRatios(ctx context.Context, layoutID int, ratios map[string]int) error {
    // Pre-check: only allow setting ratios when plan is pending
    status, err := r.planStatusByLayout(ctx, layoutID)
    if err != nil { return err }
    if status != "pending" {
        return fmt.Errorf("计划发布后不允许设置尺码比例 (layout_id=%d, status=%s)", layoutID, status)
    }

    // Start transaction
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()

    // Set flag to allow temporary zero sum during replacement
    if _, err := tx.ExecContext(ctx, `SET LOCAL cutrix.ratios_replace_flag = true`); err != nil {
        return err
    }

    // Delete existing ratios
    _, err = tx.ExecContext(ctx, `DELETE FROM production.layout_size_ratios WHERE layout_id = $1`, layoutID)
    if err != nil { return err }

    // Insert new ratios
    for size, ratio := range ratios {
        if ratio > 0 { // Only insert non-zero ratios
            _, err = tx.ExecContext(ctx,
                `INSERT INTO production.layout_size_ratios (layout_id, size, ratio) VALUES ($1, $2, $3)`,
                layoutID, size, ratio)
            if err != nil { return err }
        }
    }

    return tx.Commit()
}

// GetRatios retrieves all size ratios for a layout.
func (r *SqlLayoutsRepository) GetRatios(ctx context.Context, layoutID int) ([]models.LayoutSizeRatio, error) {
    const q = `SELECT ratio_id, layout_id, size, ratio FROM production.layout_size_ratios WHERE layout_id = $1 ORDER BY size`
    rows, err := r.db.QueryContext(ctx, q, layoutID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.LayoutSizeRatio
    for rows.Next() {
        var ratio models.LayoutSizeRatio
        if err := rows.Scan(&ratio.RatioID, &ratio.LayoutID, &ratio.Size, &ratio.Ratio); err != nil {
            return nil, err
        }
        res = append(res, ratio)
    }
    return res, rows.Err()
}