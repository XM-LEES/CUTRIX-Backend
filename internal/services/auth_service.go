package services

import (
    "context"
    "time"
)

// AuthService handles authentication, password lifecycle, and token operations.
type AuthService interface {
    // Login authenticates by name and password, validates active status, and returns tokens and user view.
    Login(ctx context.Context, name string, password string) (Tokens, *UserDTO, error)
    // Refresh validates refresh token and returns new tokens.
    Refresh(ctx context.Context, refreshToken string) (Tokens, error)
    // ChangePassword updates user's password after verifying old password.
    ChangePassword(ctx context.Context, userID int, oldPassword string, newPassword string) error
    // SetInitialPassword sets or resets the initial password without old password verification.
    SetInitialPassword(ctx context.Context, userID int, password string) error
    // ParseToken verifies a JWT token and returns claims.
    ParseToken(ctx context.Context, token string) (*Claims, error)
}

// Tokens contains access and refresh tokens and expiry.
type Tokens struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

// Claims represents verified token claims used in request context.
type Claims struct {
    UserID   int
    Name     string
    Role     string
    IsActive bool
    IssuedAt time.Time
    ExpiresAt time.Time
}