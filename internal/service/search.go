package service

import (
	"context"
	"strings"
)

func (s *SearchService) Search(ctx context.Context, query string, entityType string, limit int, offset int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}
	results, err := s.repo.Search(ctx, escapeFTSQuery(query), entityType, limit, offset)
	if err != nil {
		// Fallback for punctuation-heavy values like Jira keys if FTS parsing rejects the query.
		return s.repo.Search(ctx, `"`+strings.ReplaceAll(query, `"`, `""`)+`"`, entityType, limit, offset)
	}
	return results, err
}

func escapeFTSQuery(query string) string {
	parts := strings.Fields(query)
	for index, part := range parts {
		parts[index] = `"` + strings.ReplaceAll(part, `"`, `""`) + `"`
	}
	return strings.Join(parts, " ")
}
