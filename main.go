package traefik_ha_auth_middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Config holds the plugin configuration.
type Config struct {
	VerificationEndpoint string `json:"verificationEndpoint"`
	SourceHost           string `json:"sourceHost"`
	DestinationHost      string `json:"destinationHost"`
	EmailAddress         string `json:"emailAddress"`
}

// CreateConfig initializes the Config with default values.
func CreateConfig() *Config {
	return &Config{
		VerificationEndpoint: "",
		SourceHost:           "",
		DestinationHost:      "",
		EmailAddress:         "",
	}
}

// BearerTokenMiddleware is a middleware for verifying tokens and injecting email.
type BearerTokenMiddleware struct {
	next                 http.Handler
	verificationEndpoint string
	sourceHost           string
	destinationHost      string
	emailAddress         string
	name                 string
}

// New creates a new BearerTokenMiddleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.EmailAddress == "" {
		return nil, fmt.Errorf("emailAddress is required")
	}
	if config.SourceHost == "" || config.DestinationHost == "" {
		return nil, fmt.Errorf("sourceHost and destinationHost are required.")
	}

	return &BearerTokenMiddleware{
		next:                 next,
		verificationEndpoint: config.VerificationEndpoint,
		sourceHost:           config.SourceHost,
		destinationHost:      config.DestinationHost,
		emailAddress:         config.EmailAddress,
		name:                 name,
	}, nil
}

// ServeHTTP processes the HTTP request.
func (b *BearerTokenMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("HA_AUTH_MIDDLEWARE: Processing request for path: %s", req.URL.Path)

	// Log all cookies in the request.
	for _, cookie := range req.Cookies() {
		log.Printf("HA_AUTH_MIDDLEWARE: Cookie: %s = %s", cookie.Name, cookie.Value)
	}

	// Add the X-authentik-email header.
	req.Header.Set("X-authentik-email", b.emailAddress)
	
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

	// Replace the Host header if it matches the source host.
	if req.Host == sourceURL.Host {
		previousHost := req.Host
		req.Host = destinationURL.Host
		log.Printf("HA_AUTH_MIDDLEWARE: Host header updated: previous value: %s, new value: %s", previousHost, req.Host)
	} else {
		log.Printf("HA_AUTH_MIDDLEWARE: Host header not updated: %s, source host: %s", req.Host, sourceURL.Host)
	} 
	
	b.next.ServeHTTP(rw, req)
}

