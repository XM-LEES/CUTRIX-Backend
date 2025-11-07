package services

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"

    "golang.org/x/crypto/bcrypt"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

// authService implements AuthService using UsersRepository and HMAC-SHA256 tokens.
type authService struct {
    repo       repositories.UsersRepository
    secret     []byte
    accessTTL  time.Duration
    refreshTTL time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(repo repositories.UsersRepository, secret string, accessTTL, refreshTTL time.Duration) AuthService {
    return &authService{
        repo:       repo,
        secret:     []byte(secret),
        accessTTL:  accessTTL,
        refreshTTL: refreshTTL,
    }
}

// Login verifies credentials and returns tokens.
func (a *authService) Login(ctx context.Context, name, password string) (Tokens, *UserDTO, error) {
    u, err := a.repo.GetByName(ctx, name)
    if err != nil { return Tokens{}, nil, err }
    if u == nil || !u.IsActive { return Tokens{}, nil, ErrUnauthorized }

    if u.PasswordHash == "" { return Tokens{}, nil, ErrUnauthorized }
    if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
        return Tokens{}, nil, ErrUnauthorized
    }

    tokens, err := a.issueTokens(u)
    if err != nil { return Tokens{}, nil, err }
    return tokens, toDTO(u), nil
}

// Refresh issues new tokens based on a valid refresh token.
func (a *authService) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
    // Parse raw to check token_type, and claims for expiry.
    claims, tokenType, err := a.parseTokenRaw(refreshToken)
    if err != nil { return Tokens{}, ErrUnauthorized }
    if tokenType != "refresh" { return Tokens{}, ErrUnauthorized }
    if time.Now().After(claims.ExpiresAt) { return Tokens{}, ErrUnauthorized }

    u, err := a.repo.GetByID(ctx, claims.UserID)
    if err != nil { return Tokens{}, err }
    if u == nil || !u.IsActive { return Tokens{}, ErrUnauthorized }

    return a.issueTokens(u)
}

// ChangePassword validates old password and updates to new password.
func (a *authService) ChangePassword(ctx context.Context, userID int, oldPassword, newPassword string) error {
    if strings.TrimSpace(newPassword) == "" { return ErrValidation }

    u, err := a.repo.GetByID(ctx, userID)
    if err != nil { return err }
    if u == nil { return ErrNotFound }
    if u.PasswordHash == "" { return ErrUnauthorized }

    if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
        return ErrUnauthorized
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil { return err }
    return a.repo.UpdatePasswordHash(ctx, userID, string(hash))
}

// SetInitialPassword sets password without old password check.
func (a *authService) SetInitialPassword(ctx context.Context, userID int, newPassword string) error {
    if strings.TrimSpace(newPassword) == "" { return ErrValidation }
    _, err := a.ensureUserActive(ctx, userID)
    if err != nil { return err }
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil { return err }
    return a.repo.UpdatePasswordHash(ctx, userID, string(hash))
}

// ParseToken verifies signature and decodes claims.
func (a *authService) ParseToken(ctx context.Context, token string) (*Claims, error) {
    claims, _, err := a.parseTokenRaw(token)
    if err != nil { return nil, ErrUnauthorized }
    if time.Now().After(claims.ExpiresAt) { return nil, ErrUnauthorized }
    return claims, nil
}

// issueTokens creates access and refresh tokens.
func (a *authService) issueTokens(u *models.User) (Tokens, error) {
    now := time.Now()
    accessExp := now.Add(a.accessTTL)
    refreshExp := now.Add(a.refreshTTL)

    accessClaims := Claims{
        UserID:    u.UserID,
        Name:      u.Name,
        Role:      u.Role,
        IsActive:  u.IsActive,
        IssuedAt:  now,
        ExpiresAt: accessExp,
    }
    refreshClaims := Claims{
        UserID:    u.UserID,
        Name:      u.Name,
        Role:      u.Role,
        IsActive:  u.IsActive,
        IssuedAt:  now,
        ExpiresAt: refreshExp,
    }

    access, err := a.sign(accessClaims, "access")
    if err != nil { return Tokens{}, err }
    refresh, err := a.sign(refreshClaims, "refresh")
    if err != nil { return Tokens{}, err }

    return Tokens{AccessToken: access, RefreshToken: refresh, ExpiresAt: accessExp}, nil
}

// sign encodes claims with token type and signs with HS256.
func (a *authService) sign(c Claims, tokenType string) (string, error) {
    header := map[string]string{"alg": "HS256", "typ": "JWT"}
    headerJSON, err := json.Marshal(header)
    if err != nil { return "", err }

    // Include token_type in payload for Refresh() type check.
    payload := struct {
        Claims
        TokenType string `json:"token_type"`
    }{Claims: c, TokenType: tokenType}

    payloadJSON, err := json.Marshal(payload)
    if err != nil { return "", err }

    h := base64.RawURLEncoding.EncodeToString(headerJSON)
    p := base64.RawURLEncoding.EncodeToString(payloadJSON)
    mac := hmac.New(sha256.New, a.secret)
    mac.Write([]byte(h + "." + p))
    s := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
    return fmt.Sprintf("%s.%s.%s", h, p, s), nil
}

// parseTokenRaw verifies token and returns Claims and token_type for internal checks.
func (a *authService) parseTokenRaw(token string) (*Claims, string, error) {
    parts := strings.Split(token, ".")
    if len(parts) != 3 { return nil, "", errors.New("malformed token") }

    headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
    if err != nil { return nil, "", err }
    var header map[string]string
    if err := json.Unmarshal(headerJSON, &header); err != nil { return nil, "", err }
    if header["alg"] != "HS256" || header["typ"] != "JWT" { return nil, "", errors.New("unsupported token") }

    sig, err := base64.RawURLEncoding.DecodeString(parts[2])
    if err != nil { return nil, "", err }
    mac := hmac.New(sha256.New, a.secret)
    mac.Write([]byte(parts[0] + "." + parts[1]))
    if !hmac.Equal(sig, mac.Sum(nil)) { return nil, "", errors.New("bad signature") }

    payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil { return nil, "", err }

    // Decode claims and token_type from payload.
    var tmp struct {
        Claims
        TokenType string `json:"token_type"`
    }
    if err := json.Unmarshal(payloadJSON, &tmp); err != nil { return nil, "", err }
    return &tmp.Claims, tmp.TokenType, nil
}

func (a *authService) ensureUserActive(ctx context.Context, userID int) (*models.User, error) {
    u, err := a.repo.GetByID(ctx, userID)
    if err != nil { return nil, err }
    if u == nil { return nil, ErrNotFound }
    if !u.IsActive { return nil, ErrForbidden }
    return u, nil
}