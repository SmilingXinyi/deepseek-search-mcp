package deepseek

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	defaultSearchCount     = 10
	defaultMaxContentChars = 4000
	maxQueryChars          = 128
	maxURLChars            = 8192
	minCount               = 1
	maxCount               = 50
)

// PreparedSearch is a validated search call plus the local content cap.
type PreparedSearch struct {
	Request         SearchRequest
	MaxContentChars int
}

// PreparedFetch is a validated fetch call plus the local content cap.
type PreparedFetch struct {
	Request  FetchRequest
	MaxChars int
}

// PrepareSearch validates MCP input and builds the upstream request.
// MaxContentChars is 4000 when omitted and 0 when the caller disables the cap.
func PrepareSearch(in SearchInput) (PreparedSearch, error) {
	if strings.TrimSpace(in.Query) == "" {
		return PreparedSearch{}, fmt.Errorf("query must contain 1 to %d Unicode characters and cannot be only whitespace", maxQueryChars)
	}
	if n := utf8.RuneCountInString(in.Query); n > maxQueryChars {
		return PreparedSearch{}, fmt.Errorf("query must contain 1 to %d Unicode characters", maxQueryChars)
	}
	// Keep the caller's query, including operators and internal spacing.
	query := in.Query

	count := defaultSearchCount
	if in.Count != nil {
		count = *in.Count
		if count < minCount || count > maxCount {
			return PreparedSearch{}, fmt.Errorf("count must be between %d and %d", minCount, maxCount)
		}
	}

	ori := ""
	if in.OriQuestion != "" {
		if strings.TrimSpace(in.OriQuestion) == "" || utf8.RuneCountInString(in.OriQuestion) > maxQueryChars {
			return PreparedSearch{}, fmt.Errorf("ori_question must contain 1 to %d Unicode characters and cannot be only whitespace", maxQueryChars)
		}
		ori = in.OriQuestion
	}

	loc := strings.TrimSpace(in.Localization)
	if loc != "" && net.ParseIP(loc) == nil {
		return PreparedSearch{}, fmt.Errorf("localization must be an IP address")
	}

	maxContent := defaultMaxContentChars
	if in.MaxContentChars != nil {
		if *in.MaxContentChars < 0 {
			return PreparedSearch{}, fmt.Errorf("max_content_chars must be greater than or equal to 0")
		}
		maxContent = *in.MaxContentChars
	}

	return PreparedSearch{
		Request: SearchRequest{
			Query:        query,
			Count:        count,
			OriQuestion:  ori,
			Localization: loc,
			NeedContent:  in.NeedContent,
		},
		MaxContentChars: maxContent,
	}, nil
}

// PrepareFetch validates MCP input and builds the upstream request.
// MaxChars is 0 when omitted, which means the Markdown is not truncated.
func PrepareFetch(in FetchInput) (PreparedFetch, error) {
	raw := strings.TrimSpace(in.URL)
	if raw == "" {
		return PreparedFetch{}, fmt.Errorf("url must contain 1 to %d characters", maxURLChars)
	}
	if n := utf8.RuneCountInString(raw); n > maxURLChars {
		return PreparedFetch{}, fmt.Errorf("url must contain 1 to %d characters", maxURLChars)
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return PreparedFetch{}, fmt.Errorf("url must be an absolute http or https URL with a host and no username or password")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return PreparedFetch{}, fmt.Errorf("url must be an absolute http or https URL with a host and no username or password")
	}
	if parsed.User != nil {
		return PreparedFetch{}, fmt.Errorf("url must be an absolute http or https URL with a host and no username or password")
	}

	maxChars := 0
	if in.MaxChars != nil {
		if *in.MaxChars < 0 {
			return PreparedFetch{}, fmt.Errorf("max_chars must be greater than or equal to 0")
		}
		maxChars = *in.MaxChars
	}

	return PreparedFetch{
		Request:  FetchRequest{URL: raw},
		MaxChars: maxChars,
	}, nil
}
