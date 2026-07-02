package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
}

type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary          string   `json:"summary"`
	Description      any      `json:"description"`
	Status           Named    `json:"status"`
	Priority         Named    `json:"priority"`
	IssueType        Named    `json:"issuetype"`
	Labels           []string `json:"labels"`
	Reporter         User     `json:"reporter"`
	Assignee         User     `json:"assignee"`
	Components       []Named  `json:"components"`
	FixVersions      []Named  `json:"fixVersions"`
	ReleaseRequested []Named  `json:"-"`
	Created          string   `json:"created"`
	Updated          string   `json:"updated"`
}

type Named struct {
	Name           string          `json:"name"`
	StatusCategory *StatusCategory `json:"statusCategory,omitempty"`
}

type StatusCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
}

func New(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	return &Client{
		baseURL:  baseURL,
		email:    cfg.Email,
		apiToken: cfg.APIToken,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.email != "" && c.apiToken != ""
}

func (c *Client) BrowseURL(issueKey string) string {
	if c == nil || c.baseURL == "" {
		return ""
	}
	return c.baseURL + "/browse/" + strings.ToUpper(issueKey)
}

func (c *Client) GetIssue(ctx context.Context, issueKey string) (Issue, error) {
	if !c.Configured() {
		return Issue{}, ErrNotConfigured
	}

	requestedFields := []string{
		"summary",
		"description",
		"status",
		"priority",
		"labels",
		"issuetype",
		"created",
		"updated",
		"reporter",
		"assignee",
		"components",
		"fixVersions",
	}
	releaseRequestedFieldID, err := c.findFieldID(ctx, "Release Requested")
	if err != nil {
		return Issue{}, err
	}
	if releaseRequestedFieldID != "" {
		requestedFields = append(requestedFields, releaseRequestedFieldID)
	}
	query := url.Values{}
	query.Set("fields", strings.Join(requestedFields, ","))
	endpoint := fmt.Sprintf(
		"%s/rest/api/3/issue/%s?%s",
		c.baseURL,
		url.PathEscape(strings.ToUpper(issueKey)),
		query.Encode(),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Issue{}, err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Issue{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Issue{}, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return Issue{}, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Issue{}, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var rawIssue struct {
		ID     string          `json:"id"`
		Key    string          `json:"key"`
		Self   string          `json:"self"`
		Fields json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawIssue); err != nil {
		return Issue{}, err
	}
	var fields IssueFields
	if err := json.Unmarshal(rawIssue.Fields, &fields); err != nil {
		return Issue{}, err
	}
	if releaseRequestedFieldID != "" {
		releaseRequested, err := namedValuesFromField(rawIssue.Fields, releaseRequestedFieldID)
		if err != nil {
			return Issue{}, err
		}
		fields.ReleaseRequested = releaseRequested
	}
	return Issue{
		ID:     rawIssue.ID,
		Key:    rawIssue.Key,
		Self:   rawIssue.Self,
		Fields: fields,
	}, nil
}

func (c *Client) GetIssueID(ctx context.Context, issueKey string) (int64, error) {
	if !c.Configured() {
		return 0, ErrNotConfigured
	}
	query := url.Values{}
	query.Set("fields", "summary")
	endpoint := fmt.Sprintf(
		"%s/rest/api/3/issue/%s?%s",
		c.baseURL,
		url.PathEscape(strings.ToUpper(issueKey)),
		query.Encode(),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return 0, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return 0, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var issue struct {
		ID json.Number `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return 0, err
	}
	id, err := issue.ID.Int64()
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("jira issue id is invalid")
	}
	return id, nil
}

func (c *Client) Myself(ctx context.Context) (User, error) {
	if !c.Configured() {
		return User{}, ErrNotConfigured
	}
	endpoint := c.baseURL + "/rest/api/3/myself"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return User{}, err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return User{}, ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return User{}, fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return User{}, err
	}
	if user.AccountID == "" {
		return User{}, fmt.Errorf("jira account id is empty")
	}
	return user, nil
}

func (c *Client) findFieldID(ctx context.Context, fieldName string) (string, error) {
	endpoint := c.baseURL + "/rest/api/3/field"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil
	}

	var fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return "", err
	}
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			return field.ID, nil
		}
	}
	return "", nil
}

func namedValuesFromField(fieldsRaw json.RawMessage, fieldID string) ([]Named, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(fieldsRaw, &fields); err != nil {
		return nil, err
	}
	raw, ok := fields[fieldID]
	if !ok || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}

	var values []Named
	if err := json.Unmarshal(raw, &values); err == nil {
		return values, nil
	}

	var value Named
	if err := json.Unmarshal(raw, &value); err == nil {
		if value.Name == "" {
			return nil, nil
		}
		return []Named{value}, nil
	}

	var textValues []string
	if err := json.Unmarshal(raw, &textValues); err == nil {
		values = make([]Named, 0, len(textValues))
		for _, text := range textValues {
			if text != "" {
				values = append(values, Named{Name: text})
			}
		}
		return values, nil
	}

	var textValue string
	if err := json.Unmarshal(raw, &textValue); err == nil {
		if textValue == "" {
			return nil, nil
		}
		return []Named{{Name: textValue}}, nil
	}
	return nil, nil
}
