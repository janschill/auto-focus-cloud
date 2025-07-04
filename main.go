package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
)

var version = "dev"

// maskSensitive masks sensitive data for logging
func maskSensitive(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type SlackOAuthResponse struct {
	Ok          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Team        struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	AuthedUser struct {
		Id          string `json:"id"`
		Scope       string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	} `json:"authed_user"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP configuration missing")
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", smtpUser, to, subject, body))

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	return smtp.SendMail(addr, auth, smtpUser, []string{to}, msg)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Version:   version,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func emailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var emailReq EmailRequest
	if err := json.Unmarshal(body, &emailReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := sendEmail(emailReq.To, emailReq.Subject, emailReq.Body); err != nil {
		log.Printf("Failed to send email: %v", err)
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

func slackOAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[OAuth] Callback received: %s", r.URL.RawQuery)
	
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	
	log.Printf("[OAuth] Parameters - code: %s, state: %s, error: %s", 
		maskSensitive(code), state, errorParam)

	// Handle OAuth error
	if errorParam != "" {
		log.Printf("Slack OAuth error: %s", errorParam)
		redirectURL := fmt.Sprintf("autofocus://slack/oauth/error?error=%s", url.QueryEscape(errorParam))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Validate required parameters
	if code == "" || state == "" {
		log.Printf("Missing OAuth parameters: code=%s, state=%s", code, state)
		redirectURL := "autofocus://slack/oauth/error?error=missing_parameters"
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Exchange code for access token
	log.Printf("[OAuth] Exchanging code for token")
	tokenData, err := exchangeSlackOAuthCode(code)
	if err != nil {
		log.Printf("[OAuth] Failed to exchange code: %v", err)
		redirectURL := fmt.Sprintf("autofocus://slack/oauth/error?error=%s", url.QueryEscape("server_error"))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	if !tokenData.Ok {
		log.Printf("[OAuth] Token exchange failed: %s - %s", tokenData.Error, tokenData.ErrorDescription)
		redirectURL := fmt.Sprintf("autofocus://slack/oauth/error?error=%s", url.QueryEscape(tokenData.Error))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	log.Printf("[OAuth] Success! Team: %s, User: %s, Scopes: %s", 
		tokenData.Team.Name, tokenData.AuthedUser.Id, tokenData.Scope)

	// Show success page and redirect to app
	successHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - Slack Connected</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      text-align: center;
      padding: 50px;
      background-color: #f8f9fa;
    }
    .success {
      color: #28a745;
      font-size: 24px;
      margin-bottom: 20px;
    }
    .info {
      color: #6c757d;
      margin-bottom: 30px;
    }
    .loading {
      font-size: 14px;
      color: #007bff;
    }
  </style>
</head>
<body>
  <div class="success">✅ Slack Connected Successfully!</div>
  <div class="info">Team: %s</div>
  <div class="loading">Redirecting back to Auto-Focus...</div>
  <script>
    setTimeout(() => {
      const params = new URLSearchParams({
        access_token: '%s',
        team_id: '%s',
        team_name: '%s',
        user_id: '%s',
        scope: '%s',
        state: '%s'
      });
      window.location = 'autofocus://slack/oauth/success?' + params;
    }, 2000);
  </script>
</body>
</html>`,
		tokenData.Team.Name,
		tokenData.AccessToken,
		tokenData.Team.Id,
		tokenData.Team.Name,
		tokenData.AuthedUser.Id,
		tokenData.Scope,
		state,
	)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successHTML))
}

func exchangeSlackOAuthCode(code string) (*SlackOAuthResponse, error) {
	log.Printf("[OAuth] Starting token exchange")
	
	clientId := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	redirectURI := os.Getenv("SLACK_REDIRECT_URI")
	
	log.Printf("[OAuth] Config - ClientID: %s, RedirectURI: %s", 
		maskSensitive(clientId), redirectURI)

	if clientId == "" || clientSecret == "" {
		log.Printf("[OAuth] Missing configuration - ClientID: %t, ClientSecret: %t", 
			clientId != "", clientSecret != "")
		return nil, fmt.Errorf("slack OAuth configuration missing")
	}

	if redirectURI == "" {
		redirectURI = "https://auto-focus.app/api/slack/oauth/callback"
	}

	// Prepare form data
	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	// Make request to Slack
	log.Printf("[OAuth] Making request to Slack API")
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
	if err != nil {
		log.Printf("[OAuth] HTTP request failed: %v", err)
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[OAuth] Response status: %d", resp.StatusCode)
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[OAuth] Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	log.Printf("[OAuth] Response body: %s", string(body))

	var tokenData SlackOAuthResponse
	if err := json.Unmarshal(body, &tokenData); err != nil {
		log.Printf("[OAuth] Failed to parse JSON response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	log.Printf("[OAuth] Token exchange result - OK: %t, Error: %s", 
		tokenData.Ok, tokenData.Error)

	return &tokenData, nil
}


func main() {
	if versionBytes, err := os.ReadFile("VERSION"); err == nil {
		version = strings.TrimSpace(string(versionBytes))
	}

	godotenv.Load()

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/slack/oauth/callback", slackOAuthCallbackHandler)


	// http.HandleFunc("/api/email", emailHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Auto Focus Cloud API %s starting on port %s", version, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
