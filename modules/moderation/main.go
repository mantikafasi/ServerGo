package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server-go/common"
	"time"
)

// TypeSafe System One evaluates every field in one request and returns a
// calibrated probability per field. See https://docs.typesafe.ai/api.
const (
	moderationEndpoint = "https://api.typesafe.ai/v1/systemone"
	moderationModel    = "jev-latest"

	// requestTimeout bounds one moderation call: a reported review must not hang
	// the webhook on a slow evaluation.
	requestTimeout = 15 * time.Second
)

// moderationHTTPClient is shared so connections are reused across reports.
var moderationHTTPClient = &http.Client{Timeout: requestTimeout}

// moderationQuestions is the taxonomy in wire form, built once at startup.
var moderationQuestions = buildQuestions()

func init() {
	println("Initializing TypeSafe Moderation Service...")
}

func buildQuestions() map[string]noulQuestion {
	questions := make(map[string]noulQuestion, len(moderationFields))
	for name, f := range moderationFields {
		questions[name] = noulQuestion{
			Type:         "noul",
			Instructions: f.Instructions,
			Criteria:     &noulCriteria{True: f.Yes, False: f.No},
		}
	}
	return questions
}

// noulQuestion is one TypeSafe yes/no question.
type noulQuestion struct {
	Type         string        `json:"type"`
	Instructions string        `json:"instructions"`
	Criteria     *noulCriteria `json:"criteria,omitempty"`
}

type noulCriteria struct {
	True  string `json:"true"`
	False string `json:"false"`
}

type systemOneRequest struct {
	State     string                  `json:"state"`
	Model     string                  `json:"model"`
	Questions map[string]noulQuestion `json:"questions"`
}

type systemOneResponse struct {
	Model   string            `json:"model"`
	Answers map[string]answer `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type answer struct {
	Type string  `json:"type"`
	Noul float64 `json:"noul"`
}

// ModerationResponse represents a simplified moderation result
type ModerationResponse struct {
	Flagged    bool
	Categories map[string]bool
	Scores     map[string]float64
}

// ModerateContent analyzes content with TypeSafe, scoring every field in the
// taxonomy in a single request.
func ModerateContent(content string) (*ModerationResponse, error) {
	payload, err := json.Marshal(systemOneRequest{
		State:     content,
		Model:     moderationModel,
		Questions: moderationQuestions,
	})
	if err != nil {
		return nil, fmt.Errorf("typesafe moderation: encoding request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, moderationEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("typesafe moderation: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+common.Config.TypeSafeAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := moderationHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("typesafe moderation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("typesafe moderation: %s: %s", resp.Status, bytes.TrimSpace(body))
	}

	var parsed systemOneResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("typesafe moderation: decoding response: %w", err)
	}

	flagged := false
	categories := make(map[string]bool, len(moderationFields))
	scores := make(map[string]float64, len(moderationFields))

	for name := range moderationFields {
		score, ok := parsed.Answers[name]
		if !ok {
			return nil, fmt.Errorf("typesafe moderation: no answer for field %q", name)
		}
		if score.Type != "noul" {
			return nil, fmt.Errorf("typesafe moderation: field %q answered as %q, want noul", name, score.Type)
		}

		scores[name] = score.Noul
		categories[name] = score.Noul >= ActionThreshold

		if categories[name] {
			flagged = true
		}
	}

	return &ModerationResponse{
		Flagged:    flagged,
		Categories: categories,
		Scores:     scores,
	}, nil
}

// GetHighestScore returns the category name and score with the highest value
func GetHighestScore(response *ModerationResponse) (string, float64) {
	var highestScore float64
	var highestScoreName string

	for name, score := range response.Scores {
		if score > highestScore {
			highestScore = score
			highestScoreName = name
		}
	}

	return highestScoreName, highestScore
}
