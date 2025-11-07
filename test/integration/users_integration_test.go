package integration

import (
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestUsers_Admin_CRUD_Active(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildAuthUsersRouter(conn)

    // Seed admin and login (use unique name to avoid collisions across tests)
    admName := fmt.Sprintf("admin_%d", time.Now().UnixNano())
    _ = createUser(t, conn, admName, "admin", "Admin123!")
    wLogin, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"name":"%s","password":"Admin123!"}`, admName), "")
    if wLogin.Code != http.StatusOK { t.Fatalf("admin login code=%d body=%s", wLogin.Code, wLogin.Body.String()) }
    var loginResp struct{
        AccessToken string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresAt string `json:"expires_at"`
        User struct{ UserID int; Name string; Role string }
    }
    if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil { t.Fatalf("json login: %v", err) }

    // Create user (admin)
    wCreate, _ := doJSONAuth(r, http.MethodPost, "/api/v1/users", `{"name":"alice","role":"worker","group":"A","note":"new hire"}`, loginResp.AccessToken)
    if wCreate.Code != http.StatusCreated { t.Fatalf("create code=%d body=%s", wCreate.Code, wCreate.Body.String()) }
    var alice struct{ UserID int; Name string; Role string; IsActive bool; UserGroup *string; Note *string }
    if err := json.Unmarshal(wCreate.Body.Bytes(), &alice); err != nil { t.Fatalf("json create: %v", err) }

    // Public list by exact name should find one
    wList, _ := doJSONAuth(r, http.MethodGet, "/api/v1/users?name=alice", "", "")
    if wList.Code != http.StatusOK { t.Fatalf("list code=%d body=%s", wList.Code, wList.Body.String()) }
    var listOut []struct{ UserID int; Name string; Role string; IsActive bool; UserGroup *string; Note *string }
    if err := json.Unmarshal(wList.Body.Bytes(), &listOut); err != nil { t.Fatalf("json list: %v", err) }
    if len(listOut) != 1 || listOut[0].UserID != alice.UserID { t.Fatalf("list mismatch: %+v", listOut) }

    // Get by ID
    wGet, _ := doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", alice.UserID), "", "")
    if wGet.Code != http.StatusOK { t.Fatalf("get code=%d body=%s", wGet.Code, wGet.Body.String()) }

    // Update profile (admin)
    wUpd, _ := doJSONAuth(r, http.MethodPatch, fmt.Sprintf("/api/v1/users/%d/profile", alice.UserID), `{"name":"alice2","group":"B","note":"promoted"}`, loginResp.AccessToken)
    if wUpd.Code != http.StatusOK { t.Fatalf("update profile code=%d body=%s", wUpd.Code, wUpd.Body.String()) }
    var alice2 struct{ UserID int; Name string; Role string; IsActive bool; UserGroup *string; Note *string }
    if err := json.Unmarshal(wUpd.Body.Bytes(), &alice2); err != nil { t.Fatalf("json update: %v", err) }
    if alice2.Name != "alice2" { t.Fatalf("name not updated: %+v", alice2) }

    // Assign role (admin)
    wRole, _ := doJSONAuth(r, http.MethodPut, fmt.Sprintf("/api/v1/users/%d/role", alice2.UserID), `{"role":"manager"}`, loginResp.AccessToken)
    if wRole.Code != http.StatusNoContent { t.Fatalf("assign role code=%d body=%s", wRole.Code, wRole.Body.String()) }

    // Set password (admin)
    wPwd, _ := doJSONAuth(r, http.MethodPut, fmt.Sprintf("/api/v1/users/%d/password", alice2.UserID), `{"new_password":"Alice456!"}`, loginResp.AccessToken)
    if wPwd.Code != http.StatusNoContent { t.Fatalf("set password code=%d body=%s", wPwd.Code, wPwd.Body.String()) }

    // Login as alice2 should succeed
    wAliceLogin, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", `{"name":"alice2","password":"Alice456!"}`, "")
    if wAliceLogin.Code != http.StatusOK { t.Fatalf("alice login code=%d body=%s", wAliceLogin.Code, wAliceLogin.Body.String()) }

    // Deactivate user (admin)
    wActive, _ := doJSONAuth(r, http.MethodPut, fmt.Sprintf("/api/v1/users/%d/active", alice2.UserID), `{"active":false}`, loginResp.AccessToken)
    if wActive.Code != http.StatusNoContent { t.Fatalf("set active code=%d body=%s", wActive.Code, wActive.Body.String()) }

    // Login as alice2 should now fail
    wAliceLogin2, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", `{"name":"alice2","password":"Alice456!"}`, "")
    if wAliceLogin2.Code != http.StatusUnauthorized { t.Fatalf("inactive user should not login, code=%d", wAliceLogin2.Code) }

    // Delete user (admin)
    wDel, _ := doJSONAuth(r, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", alice2.UserID), "", loginResp.AccessToken)
    if wDel.Code != http.StatusNoContent { t.Fatalf("delete code=%d body=%s", wDel.Code, wDel.Body.String()) }

    // Get should be 404
    wGetGone, _ := doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", alice2.UserID), "", "")
    if wGetGone.Code != http.StatusNotFound { t.Fatalf("get after delete should 404, code=%d body=%s", wGetGone.Code, wGetGone.Body.String()) }
}