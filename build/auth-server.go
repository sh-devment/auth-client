package main

// auth-server.go — server-to-server calls to auth-center.
// Handles actions that originate from this app's backend, not from the browser.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// delegateCode obtains a one-time cross-app login code from auth-center,
// letting the current user open another app without re-authenticating.
// Usage: redirect the user to https://other-app/?code=<returned code>
func delegateCode(uid int64) (string, error) {
	user, err := getUserByID(uid)
	if err != nil || user == nil {
		return "", fmt.Errorf("user not found")
	}
	// Pass our stored identity through so the receiving app shows the real provider
	// and name (auth-center is stateless and keeps no profile to echo back).
	// We only store a single collapsed Name. It goes into `name`, which this
	// template's extractName reads for every method, and is mirrored into
	// first_name for older apps whose extractName reads `name` for Google only.
	// An empty name is never sent: no placeholder ("no name") is invented here,
	// so it cannot spread through the ecosystem as if it were the user's name.
	payload := map[string]string{
		"user_id":   user.AuthID,
		"app_token": appToken,
		"method":    user.Method,
	}
	if user.Name != "" {
		payload["name"] = user.Name
		payload["first_name"] = user.Name
	}
	body, _ := json.Marshal(payload)
	resp, err := httpClient.Post(authInternal+"/delegate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("delegate: %w", err)
	}
	defer resp.Body.Close()
	var data struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&data) //nolint:errcheck
	if data.Code == "" {
		return "", fmt.Errorf("delegate failed: %s", data.Error)
	}
	return data.Code, nil
}

// handleOpenApps delegates the logged-in identity to the central menu app and
// redirects the browser there with a one-time code (the "apps" menu button).
func handleOpenApps(w http.ResponseWriter, r *http.Request) {
	uid := sessionUserID(r)
	if uid == 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	code, err := delegateCode(uid)
	if err != nil {
		log.Printf("open-apps uid=%d error=%v", uid, err)
		http.Error(w, "could not open app", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "https://menu."+domain+"/?code="+code, http.StatusFound)
}
