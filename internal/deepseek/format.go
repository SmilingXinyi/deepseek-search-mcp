package deepseek

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

const truncationMarker = "\n\n[... truncated at %d characters ...]"

// SearchTool validates input, calls the Search API, and builds MCP text plus structured output.
func (c *Client) SearchTool(ctx context.Context, in SearchInput) (SearchOutput, string, error) {
	prepared, err := PrepareSearch(in)
	if err != nil {
		return SearchOutput{}, "", err
	}
	resp, err := c.Search(ctx, prepared.Request)
	if err != nil {
		return SearchOutput{}, "", err
	}

	limit := 0
	if prepared.Request.NeedContent {
		limit = prepared.MaxContentChars
	}
	results := make([]SearchResultOutput, 0, len(resp.Results))
	for _, item := range resp.Results {
		content, truncated := truncate(item.Content, limit)
		results = append(results, SearchResultOutput{
			RerankScore:      item.RerankScore,
			Title:            item.Title,
			URL:              item.URL,
			Snippet:          item.Snippet,
			Summary:          item.Summary,
			Content:          content,
			ContentTruncated: truncated,
			SiteName:         item.SiteName,
			SiteIcon:         item.SiteIcon,
			Date:             item.Date,
			Authority:        item.Authority,
		})
	}
	out := SearchOutput{Query: prepared.Request.Query, Results: results}
	return out, formatSearch(out), nil
}

// FetchTool validates input, calls the Fetch API, and builds MCP text plus structured output.
func (c *Client) FetchTool(ctx context.Context, in FetchInput) (FetchOutput, string, error) {
	prepared, err := PrepareFetch(in)
	if err != nil {
		return FetchOutput{}, "", err
	}
	resp, err := c.Fetch(ctx, prepared.Request)
	if err != nil {
		return FetchOutput{}, "", err
	}
	original := utf8.RuneCountInString(resp.Content)
	content, truncated := truncate(resp.Content, prepared.MaxChars)
	out := FetchOutput{
		URL:           prepared.Request.URL,
		Title:         resp.Title,
		Content:       content,
		Truncated:     truncated,
		OriginalChars: original,
	}
	return out, formatFetch(out), nil
}

func formatSearch(out SearchOutput) string {
	if len(out.Results) == 0 {
		return "No results found for " + out.Query + "."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Search results for %s (%d)\n", out.Query, len(out.Results))
	for i, item := range out.Results {
		fmt.Fprintf(&b, "\n%d. %s\n", i+1, item.Title)
		fmt.Fprintf(&b, "URL: %s\n", item.URL)
		if item.SiteName != nil && *item.SiteName != "" {
			fmt.Fprintf(&b, "Site: %s\n", *item.SiteName)
		}
		if item.Date != nil && *item.Date != "" {
			fmt.Fprintf(&b, "Published: %s\n", *item.Date)
		}
		fmt.Fprintf(&b, "Authority: %d\n", item.Authority)
		fmt.Fprintf(&b, "Relevance: %g\n", item.RerankScore)
		if item.Snippet != "" {
			fmt.Fprintf(&b, "Snippet: %s\n", item.Snippet)
		}
		if item.Summary != "" {
			fmt.Fprintf(&b, "Summary: %s\n", item.Summary)
		}
		if item.Content != "" {
			fmt.Fprintf(&b, "Content:\n%s\n", item.Content)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatFetch(out FetchOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\nURL: %s\n\n%s", out.Title, out.URL, out.Content)
	return b.String()
}

func truncate(s string, max int) (string, bool) {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s, false
	}
	return truncateRunes(s, max) + fmt.Sprintf(truncationMarker, max), true
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func utf8Len(s string) int {
	return utf8.RuneCountInString(s)
}
