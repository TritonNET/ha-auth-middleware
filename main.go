package ha_auth_middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Config holds the plugin configuration.
type Config struct {
	VerificationEndpoint string `json:"verificationEndpoint"`
	SourceHost           string `json:"sourceHost"`
	DestinationHost      string `json:"destinationHost"`
	DestinationHeader    string `json:"destinationHeader"`
}

// CreateConfig initializes the Config with default values.
func CreateConfig() *Config {
	return &Config{
		VerificationEndpoint: "",
		SourceHost:           "",
		DestinationHost:      "",
		DestinationHeader:    "",
	}
}

// BearerTokenMiddleware is a middleware for verifying tokens and injecting email.
type BearerTokenMiddleware struct {
	next                 http.Handler
	verificationEndpoint string
	sourceHost           string
	destinationHost      string
	destinationHeader    string
	name                 string
}

// New creates a new BearerTokenMiddleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.SourceHost == "" || config.DestinationHost == "" {
		return nil, fmt.Errorf("sourceHost and destinationHost are required")
	}
	if config.VerificationEndpoint == "" {
		return nil, fmt.Errorf("verificationEndpoint is required")
	}

	return &BearerTokenMiddleware{
		next:                 next,
		verificationEndpoint: config.VerificationEndpoint,
		sourceHost:           config.SourceHost,
		destinationHost:      config.DestinationHost,
		destinationHeader:    config.DestinationHeader,
		name:                 name,
	}, nil
}

// ServeHTTP processes the HTTP request.
func (b *BearerTokenMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check if the request is a WebSocket connection
	if req.Header.Get("Upgrade") == "websocket" {
		b.handleWebSocket(rw, req)
		return
	}

	// Extract the `haatc` cookie
	var bearerToken string
	for _, cookie := range req.Cookies() {
		if cookie.Name == "haatc" {
			bearerToken = cookie.Value
			break
		}
	}

	if bearerToken == "" {
		http.Error(rw, "Unauthorized: Missing Bearer Token", http.StatusUnauthorized)
		return
	}

	// Call the VerificationEndpoint
	email, err := b.verifyToken(bearerToken)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	var destHeader string
	if b.destinationHeader != "" {
		destHeader = b.destinationHeader
	} else {
		destHeader = "X-authentik-email"
	}

	req.Header.Set(destHeader, email)

	// Update the host if it matches the source host
	sourceURL, err := url.Parse(b.sourceHost)
	if err != nil || sourceURL.Host == "" {
		http.Error(rw, "Internal Server Error: Invalid sourceHost", http.StatusInternalServerError)
		return
	}
	destinationURL, err := url.Parse(b.destinationHost)
	if err != nil || destinationURL.Host == "" {
		http.Error(rw, "Internal Server Error: Invalid destinationHost", http.StatusInternalServerError)
		return
	}

	if req.Host == sourceURL.Host {
		req.Host = destinationURL.Host
	}

	// Pass the request to the next handler
	b.next.ServeHTTP(rw, req)
}

// handleWebSocket handles WebSocket connections.
func (b *BearerTokenMiddleware) handleWebSocket(rw http.ResponseWriter, req *http.Request) {
	// Verify the token as in the standard request
	var bearerToken string
	for _, cookie := range req.Cookies() {
		if cookie.Name == "haatc" {
			bearerToken = cookie.Value
			break
		}
	}

	if bearerToken == "" {
		http.Error(rw, "Unauthorized: Missing Bearer Token", http.StatusUnauthorized)
		return
	}

	email, err := b.verifyToken(bearerToken)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	var destHeader string
	if b.destinationHeader != "" {
		destHeader = b.destinationHeader
	} else {
		destHeader = "X-authentik-email"
	}

	req.Header.Set(destHeader, email)

	// Perform WebSocket proxying
	proxy := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			if r.Host == b.sourceHost {
				r.Host = b.destinationHost
			}
		},
	}
	proxy.ServeHTTP(rw, req)
}

// verifyToken sends the Bearer token to the VerificationEndpoint (GET) and extracts the email.
func (b *BearerTokenMiddleware) verifyToken(token string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", b.verificationEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("verification failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var responseData struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if responseData.Email == "" {
		return "", fmt.Errorf("email not found in verification response")
	}

	return responseData.Email, nil
}
