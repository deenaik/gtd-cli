package ai

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/db"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/time/rate"
)

var (
	clientOnce sync.Once
	client     *openai.Client
	limiter    *rate.Limiter
)

const (
	maxRetries    = 3
	baseBackoff   = 1 * time.Second
	ratePerMinute = 10
)

// ErrBudgetExceeded is returned when the daily token budget has been exhausted.
var ErrBudgetExceeded = errors.New("daily token budget exceeded")

// ErrNoAPIKey is returned when the OpenAI API key is not configured.
var ErrNoAPIKey = errors.New("OpenAI API key not configured; set OPENAI_API_KEY")

// Option configures a chat completion request.
type Option func(*completionConfig)

type completionConfig struct {
	maxTokens   int
	temperature *float32
	jsonSchema  *openai.ChatCompletionResponseFormatJSONSchema
}

// WithMaxTokens sets the maximum number of completion tokens.
func WithMaxTokens(n int) Option {
	return func(c *completionConfig) {
		c.maxTokens = n
	}
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float32) Option {
	return func(c *completionConfig) {
		c.temperature = &t
	}
}

// WithJSONSchema enables structured output with the given JSON schema.
func WithJSONSchema(name string, schema *openai.ChatCompletionResponseFormatJSONSchema) Option {
	return func(c *completionConfig) {
		c.jsonSchema = schema
	}
}

// GetClient returns the singleton OpenAI client, creating it from config if needed.
func GetClient() *openai.Client {
	clientOnce.Do(func() {
		cfg := config.Get()
		client = openai.NewClient(cfg.OpenAIKey)
		limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(ratePerMinute)), 1)
	})
	return client
}

// TodayUsage returns the total number of tokens used today across all features.
func TodayUsage() (int, error) {
	d := db.Get()
	var total sql.NullInt64
	err := d.QueryRow(
		`SELECT SUM(total_tokens) FROM token_usage WHERE date(created_at) = date('now')`,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("query today usage: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

// Complete sends a chat completion request with rate limiting, retry logic,
// budget enforcement, and token usage tracking.
func Complete(ctx context.Context, feature string, messages []openai.ChatCompletionMessage, opts ...Option) (string, error) {
	cfg := config.Get()
	if cfg.OpenAIKey == "" {
		return "", ErrNoAPIKey
	}

	// Check daily budget before making a request.
	used, err := TodayUsage()
	if err != nil {
		return "", fmt.Errorf("check budget: %w", err)
	}
	if used >= cfg.TokenBudget {
		return "", fmt.Errorf("%w: used %d of %d", ErrBudgetExceeded, used, cfg.TokenBudget)
	}

	// Apply options.
	cc := &completionConfig{}
	for _, o := range opts {
		o(cc)
	}

	model := cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}
	if cc.maxTokens > 0 {
		req.MaxCompletionTokens = cc.maxTokens
	}
	if cc.temperature != nil {
		req.Temperature = *cc.temperature
	}
	if cc.jsonSchema != nil {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: cc.jsonSchema,
		}
	}

	// Wait for rate limiter.
	if err := limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}

	c := GetClient()

	var resp openai.ChatCompletionResponse
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, lastErr = c.CreateChatCompletion(ctx, req)
		if lastErr == nil {
			break
		}

		if !isRetryable(lastErr) {
			return "", fmt.Errorf("openai: %w", lastErr)
		}

		if attempt == maxRetries {
			break
		}

		// Exponential backoff: 1s, 2s, 4s.
		backoff := time.Duration(float64(baseBackoff) * math.Pow(2, float64(attempt)))
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(backoff):
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("openai after %d retries: %w", maxRetries, lastErr)
	}

	// Track token usage.
	if err := trackUsage(feature, model, resp.Usage); err != nil {
		// Log but do not fail the request for a tracking error.
		_ = err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("openai: empty response, no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// isRetryable determines whether an API error warrants a retry.
// Retries on HTTP 429 (rate limit) and 5xx (server errors).
func isRetryable(err error) bool {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPStatusCode == 429 {
			return true
		}
		if apiErr.HTTPStatusCode >= 500 && apiErr.HTTPStatusCode < 600 {
			return true
		}
	}
	// Also retry on transient network-like errors indicated in the message.
	if strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "timeout") {
		return true
	}
	return false
}

// trackUsage inserts a token usage record into the database.
func trackUsage(feature, model string, usage openai.Usage) error {
	d := db.Get()
	_, err := d.Exec(
		`INSERT INTO token_usage (feature, model, prompt_tokens, completion_tokens, total_tokens)
		 VALUES (?, ?, ?, ?, ?)`,
		feature, model, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
	)
	return err
}
