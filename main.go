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
}

// CreateConfig initializes the Config with default values.
func CreateConfig() *Config {
	return &Config{
		VerificationEndpoint: "",
	}
}

// BearerTokenMiddleware is a middleware for verifying Bearer tokens and injecting email.
type BearerTokenMiddleware struct {
	next                 http.Handler
	verificationEndpoint string
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
		name:                 name,
	}, nil
}

// ServeHTTP processes the HTTP request.
func (b *BearerTokenMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(rw, "missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	email, err := b.verifyTokenAndGetEmail(token)
	if err != nil {
		http.Error(rw, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Inject the email as a header into the request.
	req.Header.Set("X-Authentik-Email", email)

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
