package jira

import "errors"

var (
	ErrNotConfigured = errors.New("jira integration is not configured")
	ErrUnauthorized  = errors.New("jira authentication failed")
	ErrNotFound      = errors.New("jira issue not found")
)
