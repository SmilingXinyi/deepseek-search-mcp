package deepseek

// SearchInput is the MCP input for deepseek_search.
// MaxContentChars is applied locally and is not sent to the DeepSeek API.
type SearchInput struct {
	Query           string `json:"query" jsonschema:"Search query, 1 to 128 Unicode characters, not all whitespace. Operators such as site:, -site:, lang:, after:, and before: are forwarded as-is."`
	Count           *int   `json:"count,omitempty" jsonschema:"Number of results to return, from 1 to 50. Defaults to 10."`
	OriQuestion     string `json:"ori_question,omitempty" jsonschema:"Original user question that triggered the search, 1 to 128 characters. Improves query understanding."`
	Localization    string `json:"localization,omitempty" jsonschema:"Optional IP address used to localize results. Send only when the user provided it."`
	NeedContent     bool   `json:"need_content,omitempty" jsonschema:"When true, include page content on each result. Content is empty when false."`
	MaxContentChars *int   `json:"max_content_chars,omitempty" jsonschema:"Local cap on each result content field when need_content is true. Defaults to 4000. 0 means no limit."`
}

// SearchRequest is the JSON body for POST /v1/tools/search.
type SearchRequest struct {
	Query        string `json:"query"`
	Count        int    `json:"count"`
	OriQuestion  string `json:"ori_question,omitempty"`
	Localization string `json:"localization,omitempty"`
	NeedContent  bool   `json:"need_content,omitempty"`
}

// SearchResult is one item from the Search API.
type SearchResult struct {
	RerankScore float64 `json:"rerank_score"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Summary     string  `json:"summary"`
	Content     string  `json:"content"`
	SiteName    *string `json:"site_name"`
	SiteIcon    *string `json:"site_icon"`
	Date        *string `json:"date"`
	Authority   int     `json:"authority"`
}

// SearchResponse is the Search API response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResultOutput is one MCP search result after local truncation.
type SearchResultOutput struct {
	RerankScore      float64 `json:"rerank_score" jsonschema:"Relevance score from 0 to 3. 1.5 or above is high quality."`
	Title            string  `json:"title" jsonschema:"Result title."`
	URL              string  `json:"url" jsonschema:"Result URL."`
	Snippet          string  `json:"snippet" jsonschema:"Short summary for display."`
	Summary          string  `json:"summary" jsonschema:"Longer summary intended for a model."`
	Content          string  `json:"content" jsonschema:"Page content. Empty when need_content is false or content is unavailable."`
	ContentTruncated bool    `json:"content_truncated,omitempty" jsonschema:"True when content was truncated locally."`
	SiteName         *string `json:"site_name" jsonschema:"Site name, or null when unavailable."`
	SiteIcon         *string `json:"site_icon" jsonschema:"Site icon URL, or null when unavailable."`
	Date             *string `json:"date" jsonschema:"Publication time in RFC 3339 UTC, or null when unknown."`
	Authority        int     `json:"authority" jsonschema:"Authority tier from 0 to 4. -1 means unknown. 3 or above is highly authoritative."`
}

// SearchOutput is the structured MCP result for deepseek_search.
type SearchOutput struct {
	Query   string               `json:"query" jsonschema:"Query that was searched."`
	Results []SearchResultOutput `json:"results" jsonschema:"Ranked results. Empty when the query matches nothing."`
}

// FetchInput is the MCP input for deepseek_fetch.
// MaxChars is applied locally and is not sent to the DeepSeek API.
type FetchInput struct {
	URL      string `json:"url" jsonschema:"Absolute http or https URL to fetch, at most 8192 characters, with a host and no username or password."`
	MaxChars *int   `json:"max_chars,omitempty" jsonschema:"Local cap on the Markdown content. Defaults to 0, which means no truncation."`
}

// FetchRequest is the JSON body for POST /v1/tools/fetch.
type FetchRequest struct {
	URL string `json:"url"`
}

// FetchResponse is the Fetch API response.
type FetchResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// FetchOutput is the structured MCP result for deepseek_fetch.
type FetchOutput struct {
	URL           string `json:"url" jsonschema:"URL that was fetched."`
	Title         string `json:"title" jsonschema:"Page title. Empty when the title is unavailable."`
	Content       string `json:"content" jsonschema:"Page content as Markdown."`
	Truncated     bool   `json:"truncated" jsonschema:"True when content was truncated locally."`
	OriginalChars int    `json:"original_chars" jsonschema:"Unicode character count of the content before local truncation."`
}
