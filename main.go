package traefik_ha_auth_middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v2/pkg/plugins"
)

// Config holds the plugin configuration.
type Config struct {
	VerificationEndpoint string `json:"verificationEndpoint"`
	SourcePath           string `json:"sourcePath"`
	DestinationPath      string `json:"destinationPath"`
	EmailAddress	     string `json:"emailAddress"`
}

// CreateConfig initializes the Config with default values.
func CreateConfig() *Config {
	return &Config{
		VerificationEndpoint: "",
		SourcePath:           "",
		DestinationPath:      "",
		EmailAddress:         "",
	}
}

// BearerTokenMiddleware is a middleware for verifying tokens and injecting email.
type BearerTokenMiddleware struct {
	next                 http.Handler
	verificationEndpoint string
	sourcePath           string
	destinationPath      string
	emailAddress	     string
	name                 string
}

// New creates a new BearerTokenMiddleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &BearerTokenMiddleware{
		next:                 next,
		verificationEndpoint: config.VerificationEndpoint,
		sourcePath:           config.SourcePath,
		destinationPath:      config.DestinationPath,
		emailAddress:         config.EmailAddress,
		name:                 name,
	}, nil
}

// ServeHTTP processes the HTTP request.
func (b *BearerTokenMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Log all cookies in the request.
	for _, cookie := range req.Cookies() {
		log.Printf("Cookie: %s = %s", cookie.Name, cookie.Value)
	}

	req.Header.Set("X-authentik-email", b.emailAddress)

	// Replace the request path if it matches the source path.
	if strings.HasPrefix(req.URL.Path, b.sourcePath) {
		req.URL.Path = strings.Replace(req.URL.Path, b.sourcePath, b.destinationPath, 1)
	}

	// Pass the request to the next handler.
	b.next.ServeHTTP(rw, req)
}