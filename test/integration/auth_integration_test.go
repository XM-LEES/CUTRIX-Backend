package integration

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    "strings"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
)

// buildAuthUsersRouter registers only finalized auth and users routes.
func buildAuthUsersRouter(conn *sql.DB) *gin.Engine {
    r := gin.New()
    r.Use(middleware.RequestID())
    api := r.Group("/api/v1")

    repo := repositories.NewSqlUsersRepository(conn)
    authSvc := services.NewAuthService(repo, "test-secret", time.Minute, 24*time.Hour)
    usersSvc := services.NewUsersService(repo)
    handlers.RegisterRoutes(api, authSvc, usersSvc)
    return r
}

// createUser inserts a user with a hashed password.
func createUser(t *testing.T, conn *sql.DB, name, role, password string) int {
    t.Helper()
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil { t.Fatalf("bcrypt: %v", err) }
    var id int
    err = conn.QueryRow(`INSERT INTO public.users (name, password_hash, role, is_active) VALUES ($1, $2, $3, TRUE) RETURNING user_id`, name, string(hash), role).Scan(&id)
    if err != nil { t.Fatalf("insert user: %v", err) }
    return id
}

func TestAuth_Login_Refresh_ChangePassword(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildAuthUsersRouter(conn)

    // Seed admin user with unique name
    admName := fmt.Sprintf("admin_%d", time.Now().UnixNano())
    _ = createUser(t, conn, admName, "admin", "Admin123!")

    // Login
    w1, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"name":"%s","password":"Admin123!"}`, admName), "")
    if w1.Code != http.StatusOK { t.Fatalf("login code=%d body=%s", w1.Code, w1.Body.String()) }
    var loginResp struct{
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresAt    string `json:"expires_at"`
        User         struct{ UserID int; Name string; Role string }
    }
    if err := json.Unmarshal(w1.Body.Bytes(), &loginResp); err != nil { t.Fatalf("json login: %v", err) }

    // Refresh
    w2, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"`+loginResp.RefreshToken+`"}`, "")
    if w2.Code != http.StatusOK { t.Fatalf("refresh code=%d body=%s", w2.Code, w2.Body.String()) }

    // Change password (authorized)
    body := `{"old_password":"Admin123!","new_password":"Admin456!"}`
    req, _ := http.NewRequest(http.MethodPut, "/api/v1/auth/password", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
    w3 := httptest.NewRecorder()
    r.ServeHTTP(w3, req)
    if w3.Code != http.StatusNoContent { t.Fatalf("change password code=%d body=%s", w3.Code, w3.Body.String()) }

    // Old password should fail
    w4, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"name":"%s","password":"Admin123!"}`, admName), "")
    if w4.Code != http.StatusUnauthorized { t.Fatalf("old password should fail, got code=%d", w4.Code) }

    // New password should succeed
    w5, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"name":"%s","password":"Admin456!"}`, admName), "")
    if w5.Code != http.StatusOK { t.Fatalf("new password login code=%d body=%s", w5.Code, w5.Body.String()) }
}