package tempo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("tempo integration is not configured")
	ErrUnauthorized  = errors.New("tempo authentication failed")
)

type Config struct {
	BaseURL  string
	APIToken string
}

type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

type WorklogInput struct {
	AuthorAccountID string `json:"authorAccountId"`
	IssueID         int64  `json:"issueId"`
	TimeSpent       int64  `json:"timeSpentSeconds"`
	BillableSeconds int64  `json:"billableSeconds,omitempty"`
	StartDate       string `json:"startDate"`
	StartTime       string `json:"startTime,omitempty"`
	Description     string `json:"description,omitempty"`
}

type Worklog struct {
	TempoWorklogID   int64  `json:"tempoWorklogId"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
	BillableSeconds  int64  `json:"billableSeconds"`
	StartDate        string `json:"startDate"`
	StartTime        string `json:"startTime"`
	Description      string `json:"description"`
	Self             string `json:"self"`
	Issue            struct {
		ID  int64  `json:"id"`
		Key string `json:"key"`
	} `json:"issue"`
}

type WorklogFilter struct {
	From   string
	To     string
	Limit  int
	Offset int
}

type worklogPage struct {
	Results  []Worklog `json:"results"`
	Metadata struct {
		Count int    `json:"count"`
		Next  string `json:"next"`
	} `json:"metadata"`
}

func New(cfg Config) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	baseURL = strings.TrimSuffix(baseURL, "/4")
	return &Client{
		baseURL:  baseURL,
		apiToken: strings.TrimSpace(cfg.APIToken),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.apiToken != ""
}

func (c *Client) CreateWorklog(ctx context.Context, input WorklogInput) (Worklog, error) {
	if !c.Configured() {
		return Worklog{}, ErrNotConfigured
	}
	data, err := json.Marshal(input)
	if err != nil {
		return Worklog{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/4/worklogs", bytes.NewReader(data))
	if err != nil {
		return Worklog{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Worklog{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Worklog{}, ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Worklog{}, fmt.Errorf("tempo returned status %d: %s", resp.StatusCode, responseSnippet(resp.Body))
	}

	var worklog Worklog
	if err := json.NewDecoder(resp.Body).Decode(&worklog); err != nil {
		return Worklog{}, err
	}
	return worklog, nil
}

func (c *Client) ListUserWorklogs(ctx context.Context, accountID string, filter WorklogFilter) ([]Worklog, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	worklogs := []Worklog{}
	for {
		query := url.Values{}
		query.Set("from", filter.From)
		query.Set("to", filter.To)
		query.Set("limit", fmt.Sprint(limit))
		query.Set("offset", fmt.Sprint(offset))
		endpoint := fmt.Sprintf("%s/4/worklogs/user/%s?%s", c.baseURL, url.PathEscape(accountID), query.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			_ = resp.Body.Close()
			return nil, ErrUnauthorized
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			message := responseSnippet(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("tempo returned status %d: %s", resp.StatusCode, message)
		}
		var page worklogPage
		err = json.NewDecoder(resp.Body).Decode(&page)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		worklogs = append(worklogs, page.Results...)
		if page.Metadata.Next == "" || len(page.Results) == 0 {
			break
		}
		offset += limit
	}
	return worklogs, nil
}

func responseSnippet(reader io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(reader, 1024))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
