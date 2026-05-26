package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// providerVersion identifies the terraform-provider-hivehook build in the
// outbound User-Agent. Releases override this at build time via -ldflags
// "-X github.com/hivehook/terraform-provider-hivehook/internal/provider.providerVersion=<vX.Y.Z>".
var providerVersion = "dev"

// userAgent returns the canonical User-Agent header value the client sends on
// every request. The format follows the convention used by other HashiCorp
// providers: "terraform-provider-<name>/<version>".
func userAgent() string {
	return "terraform-provider-hivehook/" + providerVersion
}

// Retry tuning. Kept as package-level constants so the values are documented
// in one place and easy to flip if the server's behaviour changes.
const (
	defaultHTTPTimeout = 30 * time.Second
	maxRetries         = 4
	retryBaseDelay     = 200 * time.Millisecond
	retryMaxDelay      = 5 * time.Second
)

// Client is a thin GraphQL transport with bounded retries and a few production
// niceties (User-Agent, per-request context deadline, Retry-After honouring).
type Client struct {
	url    string
	apiKey string
	http   *http.Client
}

// NewClient builds a Client bound to the given GraphQL server. The supplied
// base URL is normalised so callers can pass either "https://h.example.com"
// or "https://h.example.com/" interchangeably.
func NewClient(url, apiKey string) *Client {
	return &Client{
		url:    strings.TrimRight(url, "/") + "/graphql",
		apiKey: apiKey,
		http:   &http.Client{Timeout: defaultHTTPTimeout},
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

// Execute issues a single GraphQL request, retrying transient failures
// (connect errors, 429, 5xx) with bounded exponential backoff. The caller's
// context governs the overall deadline; if the context expires mid-retry the
// in-flight error is returned immediately.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]any, result any) error {
	body, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			if lastErr != nil {
				return fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		}

		respBody, status, retryAfter, doErr := c.do(ctx, body)
		if doErr != nil {
			if !isRetryableNetErr(doErr) || attempt == maxRetries {
				return fmt.Errorf("executing request: %w", doErr)
			}
			lastErr = doErr
			if err := sleep(ctx, backoffDelay(attempt)); err != nil {
				return err
			}
			continue
		}

		if status == http.StatusOK {
			return decodeGQL(respBody, result)
		}

		if isRetryableStatus(status) && attempt < maxRetries {
			lastErr = fmt.Errorf("unexpected status %d: %s", status, truncate(respBody))
			delay := retryAfter
			if delay <= 0 {
				delay = backoffDelay(attempt)
			}
			if err := sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}

		return fmt.Errorf("unexpected status %d: %s", status, truncate(respBody))
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("hivehook client: exhausted retries")
}

// do issues a single HTTP round trip and returns the response body, status
// code, and a parsed Retry-After delay (if the server sent one). Wire-level
// errors (connect failure, TLS issues) are returned as the fourth value.
func (c *Client) do(ctx context.Context, body []byte) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, resp.StatusCode, 0, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, parseRetryAfter(resp.Header.Get("Retry-After")), nil
}

// decodeGQL unpacks a GraphQL envelope: it checks for top-level errors and,
// when result is non-nil, unmarshals the `data` field into it.
func decodeGQL(body []byte, result any) error {
	var gqlResp gqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, len(gqlResp.Errors))
		for i, e := range gqlResp.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("graphql error: %s", strings.Join(msgs, "; "))
	}
	if result != nil {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("unmarshaling data: %w", err)
		}
	}
	return nil
}

// isRetryableStatus reports whether an HTTP status code identifies a
// transient server-side condition. 5xx covers server faults; 429 covers rate
// limiting (with a Retry-After hint we honour separately).
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// isRetryableNetErr reports whether an outbound transport error is worth
// retrying. We retry connection-level problems and timeouts; we do not retry
// canceled contexts (the caller asked us to stop).
func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return true
}

// backoffDelay returns the exponential-backoff delay for the given attempt
// index, capped at retryMaxDelay. Attempt 0 yields retryBaseDelay.
func backoffDelay(attempt int) time.Duration {
	d := retryBaseDelay << attempt
	if d > retryMaxDelay {
		return retryMaxDelay
	}
	return d
}

// parseRetryAfter interprets the Retry-After header per RFC 7231: an integer
// seconds value or an HTTP-date. Unparseable or absent headers return 0,
// signalling "use the default backoff".
func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// sleep blocks for d but returns early (and reports the cancellation) when
// the context is canceled. Using a timer instead of time.Sleep keeps the
// retry loop responsive to deadlines.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// truncate returns the response body as a string, trimmed to a manageable
// length so a misbehaving server cannot blow up our error messages.
func truncate(body []byte) string {
	const max = 512
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "...(truncated)"
}
