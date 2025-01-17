package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v2/pkg/plugins"
)

// Config holds the plugin configuration.
type Config struct {
	VerificationEndpoint string `json:"verificationEndpoint"`
	SourcePath           string `json:"sourcePath"`
	DestinationPath      string `json:"destinationPath"`
}

// CreateConfig initializes the Config with default values.
func CreateConfig() *Config {
	return &Config{
		VerificationEndpoint: "",
		SourcePath:           "",
		DestinationPath:      "",
	}
}

// BearerTokenMiddleware is a middleware for verifying tokens and injecting email.
type BearerTokenMiddleware struct {
	next                 http.Handler
	verificationEndpoint string
	sourcePath           string
	destinationPath      string
	name                 string
}

// New creates a new BearerTokenMiddleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.VerificationEndpoint == "" {
		return nil, fmt.Errorf("verificationEndpoint is required")
	}

	return &BearerTokenMiddleware{
		next:                 next,
		verificationEndpoint: config.VerificationEndpoint,
		sourcePath:           config.SourcePath,
		destinationPath:      config.DestinationPath,
		name:                 name,
	}, nil
}

// ServeHTTP processes the HTTP request.
func (b *BearerTokenMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Extract the token from the "auth_token" cookie.
	cookie, err := req.Cookie("auth_token")
	if err != nil {
		http.Error(rw, "missing or invalid auth_token cookie", http.StatusUnauthorized)
		return
	}
	
	// Verify the token and get the email.
	email, err := b.verifyTokenAndGetEmail(cookie.Value)
	if err != nil {
		http.Error(rw, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Inject the email as a header into the request.
	req.Header.Set("X-Authentik-Email", email)

	// Replace the request path if it matches the source path.
	if strings.HasPrefix(req.URL.Path, b.sourcePath) {
		req.URL.Path = strings.Replace(req.URL.Path, b.sourcePath, b.destinationPath, 1)
	}

	// Pass the request to the next handler.
	b.next.ServeHTTP(rw, req)
}

// verifyTokenAndGetEmail calls the external endpoint to validate the token and retrieve the email.
func (b *BearerTokenMiddleware) verifyTokenAndGetEmail(token string) (string, error) {
	payload, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	resp, err := http.Post(b.verificationEndpoint, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("verification failed: %s", string(body))
	}

	// Parse the response body to extract the email.
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
