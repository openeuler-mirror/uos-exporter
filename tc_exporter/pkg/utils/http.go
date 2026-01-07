package utils

import (
	"fmt"
	"net/url"
)

func ValidateURI(uri string) error {
	parsedURL, err := url.ParseRequestURI(uri)
	if err != nil {
		return fmt.Errorf("invalid URI format: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}
	return nil
}
