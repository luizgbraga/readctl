package research

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Source      string `json:"source"` // "web", "academic", "book"
	PublishedAt string `json:"published_at,omitempty"`
	Authors     string `json:"authors,omitempty"`
	Citations   int    `json:"citations,omitempty"`
}

type SearchClient struct {
	firecrawlAPIKey string
	httpClient      *http.Client
}

func NewSearchClient(firecrawlAPIKey string) *SearchClient {
	return &SearchClient{
		firecrawlAPIKey: firecrawlAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SearchWeb performs a web search using Firecrawl Search API
func (c *SearchClient) SearchWeb(ctx context.Context, query string, count int) ([]SearchResult, error) {
	if c.firecrawlAPIKey == "" {
		return nil, fmt.Errorf("firecrawl API key not configured")
	}

	if count <= 0 {
		count = 5
	}

	endpoint := "https://api.firecrawl.dev/v1/search"
	reqBody := firecrawlSearchRequest{
		Query: query,
		Limit: count,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.firecrawlAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firecrawl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("firecrawl API error (status %d): %s", resp.StatusCode, string(body))
	}

	var fcResp firecrawlSearchResponse
	if err := json.Unmarshal(body, &fcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(body))
	}

	var results []SearchResult
	for _, r := range fcResp.Data {
		snippet := r.Description
		if snippet == "" && r.Markdown != "" {
			snippet = r.Markdown
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
		}
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: snippet,
			Source:  "web",
		})
	}

	return results, nil
}

// SearchAcademic searches for academic papers using Semantic Scholar API
func (c *SearchClient) SearchAcademic(ctx context.Context, query string, count int) ([]SearchResult, error) {
	if count <= 0 {
		count = 5
	}

	endpoint := "https://api.semanticscholar.org/graph/v1/paper/search"
	params := url.Values{}
	params.Set("query", query)
	params.Set("limit", fmt.Sprintf("%d", count))
	params.Set("fields", "title,url,abstract,authors,year,citationCount")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("academic search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("semantic scholar API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ssResp semanticScholarResponse
	if err := json.NewDecoder(resp.Body).Decode(&ssResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var results []SearchResult
	for _, p := range ssResp.Data {
		var authors []string
		for _, a := range p.Authors {
			authors = append(authors, a.Name)
		}

		snippet := p.Abstract
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}

		results = append(results, SearchResult{
			Title:       p.Title,
			URL:         p.URL,
			Snippet:     snippet,
			Source:      "academic",
			PublishedAt: fmt.Sprintf("%d", p.Year),
			Authors:     strings.Join(authors, ", "),
			Citations:   p.CitationCount,
		})
	}

	return results, nil
}

// SearchBookContext searches for book-specific content (reviews, analysis, excerpts)
func (c *SearchClient) SearchBookContext(ctx context.Context, bookTitle, author, topic string, count int) ([]SearchResult, error) {
	// Build targeted queries for literary analysis
	queries := []string{
		fmt.Sprintf(`"%s" %s analysis literary criticism`, bookTitle, topic),
		fmt.Sprintf(`"%s" %s chapter summary interpretation`, bookTitle, topic),
		fmt.Sprintf(`%s "%s" scholarly review`, author, bookTitle),
	}

	var allResults []SearchResult
	perQuery := (count + len(queries) - 1) / len(queries)

	for _, q := range queries {
		results, err := c.SearchWeb(ctx, q, perQuery)
		if err != nil {
			continue // Skip failed queries
		}
		allResults = append(allResults, results...)
		if len(allResults) >= count {
			break
		}
	}

	// Mark as book-related
	for i := range allResults {
		allResults[i].Source = "book"
	}

	if len(allResults) > count {
		allResults = allResults[:count]
	}

	return allResults, nil
}

// Firecrawl Search API request/response structures
type firecrawlSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type firecrawlSearchResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Markdown    string `json:"markdown,omitempty"`
	} `json:"data"`
}

// Semantic Scholar API response structures
type semanticScholarResponse struct {
	Data []struct {
		PaperID       string `json:"paperId"`
		Title         string `json:"title"`
		URL           string `json:"url"`
		Abstract      string `json:"abstract"`
		Year          int    `json:"year"`
		CitationCount int    `json:"citationCount"`
		Authors       []struct {
			Name string `json:"name"`
		} `json:"authors"`
	} `json:"data"`
}
