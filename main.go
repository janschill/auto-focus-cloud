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
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

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
	tokenData, err := exchangeSlackOAuthCode(code)
	if err != nil {
		log.Printf("Failed to exchange OAuth code: %v", err)
		redirectURL := fmt.Sprintf("autofocus://slack/oauth/error?error=%s", url.QueryEscape("server_error"))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	if !tokenData.Ok {
		log.Printf("Slack OAuth token exchange failed: %s", tokenData.Error)
		redirectURL := fmt.Sprintf("autofocus://slack/oauth/error?error=%s", url.QueryEscape(tokenData.Error))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

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
	clientId := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")
	redirectURI := os.Getenv("SLACK_REDIRECT_URI")

	if clientId == "" || clientSecret == "" {
		return nil, fmt.Errorf("Slack OAuth configuration missing")
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
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenData SlackOAuthResponse
	if err := json.Unmarshal(body, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tokenData, nil
}

func slackOAuthTestHandler(w http.ResponseWriter, r *http.Request) {
	clientId := os.Getenv("SLACK_CLIENT_ID")
	if clientId == "" {
		http.Error(w, "SLACK_CLIENT_ID not configured", http.StatusInternalServerError)
		return
	}

	// Generate a test state parameter
	state := fmt.Sprintf("test-%d", time.Now().Unix())
	scopes := "users.profile:read,users.profile:write,dnd:write"
	redirectURI := os.Getenv("SLACK_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "https://auto-focus.app/api/slack/oauth/callback"
	}

	authURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&redirect_uri=%s&state=%s&user_scope=%s",
		url.QueryEscape(clientId),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(scopes),
	)

	testHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - Test Slack OAuth</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 600px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    h1 { color: #333; margin-bottom: 20px; }
    .info {
      background: #e7f3ff;
      padding: 15px;
      border-radius: 6px;
      margin: 20px 0;
      border-left: 4px solid #007bff;
    }
    .button {
      display: inline-block;
      background: #4a154b;
      color: white;
      padding: 12px 24px;
      text-decoration: none;
      border-radius: 6px;
      font-weight: bold;
      margin: 10px 0;
    }
    .button:hover { background: #611f69; }
    .details {
      font-family: monospace;
      background: #f8f9fa;
      padding: 10px;
      border-radius: 4px;
      margin: 10px 0;
      font-size: 12px;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>Auto-Focus Slack OAuth Test</h1>

    <div class="info">
      <strong>Testing Instructions:</strong><br>
      1. Click the "Connect to Slack" button below<br>
      2. You'll be redirected to Slack for authorization<br>
      3. After approval, you'll see the OAuth response data<br>
      4. Check server logs for any errors
    </div>

    <a href="%s" class="button">Connect to Slack</a>

    <div class="details">
      <strong>OAuth Details:</strong><br>
      Client ID: %s<br>
      Redirect URI: %s<br>
      Scopes: %s<br>
      State: %s
    </div>

    <div class="info">
      <strong>Note:</strong> This is a test endpoint. In production, your macOS app will handle the OAuth flow.
    </div>
  </div>
</body>
</html>`,
		authURL,
		clientId,
		redirectURI,
		scopes,
		state,
	)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(testHTML))
}

func slackOAuthCallbackTestHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// For testing, show the results instead of redirecting
	var resultHTML string

	if errorParam != "" {
		resultHTML = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - OAuth Error</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 600px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    .error { color: #dc3545; font-size: 24px; margin-bottom: 20px; }
    .details {
      font-family: monospace;
      background: #f8f9fa;
      padding: 15px;
      border-radius: 4px;
      margin: 20px 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="error">❌ OAuth Error</div>
    <div class="details">Error: %s</div>
    <a href="/slack/test">← Try Again</a>
  </div>
</body>
</html>`, errorParam)
	} else if code == "" || state == "" {
		resultHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - OAuth Error</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 600px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    .error { color: #dc3545; font-size: 24px; margin-bottom: 20px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="error">❌ Missing Parameters</div>
    <p>OAuth code or state parameter is missing.</p>
    <a href="/slack/test">← Try Again</a>
  </div>
</body>
</html>`
	} else {
		// Exchange code for token
		tokenData, err := exchangeSlackOAuthCode(code)
		if err != nil {
			resultHTML = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - OAuth Error</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 600px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    .error { color: #dc3545; font-size: 24px; margin-bottom: 20px; }
    .details {
      font-family: monospace;
      background: #f8f9fa;
      padding: 15px;
      border-radius: 4px;
      margin: 20px 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="error">❌ Token Exchange Failed</div>
    <div class="details">Error: %s</div>
    <a href="/slack/test">← Try Again</a>
  </div>
</body>
</html>`, err.Error())
		} else if !tokenData.Ok {
			resultHTML = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - OAuth Error</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 600px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    .error { color: #dc3545; font-size: 24px; margin-bottom: 20px; }
    .details {
      font-family: monospace;
      background: #f8f9fa;
      padding: 15px;
      border-radius: 4px;
      margin: 20px 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="error">❌ Slack OAuth Failed</div>
    <div class="details">Error: %s</div>
    <a href="/slack/test">← Try Again</a>
  </div>
</body>
</html>`, tokenData.Error)
		} else {
			// Success - show the token data
			resultHTML = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <title>Auto-Focus - OAuth Success</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, sans-serif;
      max-width: 800px;
      margin: 50px auto;
      padding: 20px;
      background-color: #f8f9fa;
    }
    .container {
      background: white;
      padding: 30px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    .success { color: #28a745; font-size: 24px; margin-bottom: 20px; }
    .info {
      background: #d4edda;
      padding: 15px;
      border-radius: 6px;
      margin: 20px 0;
      border-left: 4px solid #28a745;
    }
    .data {
      font-family: monospace;
      background: #f8f9fa;
      padding: 15px;
      border-radius: 4px;
      margin: 20px 0;
      font-size: 12px;
      overflow-x: auto;
    }
    .warning {
      background: #fff3cd;
      padding: 15px;
      border-radius: 6px;
      margin: 20px 0;
      border-left: 4px solid #ffc107;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="success">✅ OAuth Success!</div>

    <div class="info">
      <strong>Team Connected:</strong> %s<br>
      <strong>User ID:</strong> %s<br>
      <strong>Scopes:</strong> %s
    </div>

    <div class="warning">
      <strong>⚠️ Security Note:</strong> This test page shows sensitive data. In production, tokens are securely passed to your macOS app.
    </div>

    <details>
      <summary><strong>🔍 Full OAuth Response Data (Click to expand)</strong></summary>
      <div class="data">
Access Token: %s<br>
Team ID: %s<br>
Team Name: %s<br>
User ID: %s<br>
User Access Token: %s<br>
Scope: %s<br>
State: %s<br>
Token Type: %s
      </div>
    </details>

    <p><a href="/slack/test">← Test Again</a></p>
  </div>
</body>
</html>`,
				tokenData.Team.Name,
				tokenData.AuthedUser.Id,
				tokenData.Scope,
				tokenData.AccessToken,
				tokenData.Team.Id,
				tokenData.Team.Name,
				tokenData.AuthedUser.Id,
				tokenData.AuthedUser.AccessToken,
				tokenData.Scope,
				state,
				tokenData.TokenType,
			)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resultHTML))
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

	// Test endpoints (remove in production)
	http.HandleFunc("/slack/test", slackOAuthTestHandler)
	http.HandleFunc("/slack/oauth/callback-test", slackOAuthCallbackTestHandler)

	// http.HandleFunc("/api/email", emailHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Auto Focus Cloud API %s starting on port %s", version, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
