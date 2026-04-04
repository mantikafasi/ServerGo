package moderation

import (
	"context"
	"server-go/common"

	openai "github.com/sashabaranov/go-openai"
)

var moderationClient *openai.Client

func init() {
	println("Initializing OpenAI Moderation Service...")
	moderationClient = openai.NewClient(common.Config.OpenAIModerationAPIKey)
}

// ModerationResponse represents a simplified moderation result
type ModerationResponse struct {
	Flagged    bool
	Categories map[string]bool
	Scores     map[string]float64
}

// ModerateContent analyzes content using OpenAI's moderation API
func ModerateContent(content string) (*ModerationResponse, error) {
	ctx := context.Background()

	req := openai.ModerationRequest{
		Model: openai.ModerationTextLatest,
		Input: content,
	}

	resp, err := moderationClient.Moderations(ctx, req)
	if err != nil {
		return nil, err
	}

	// OpenAI returns an array of results, we'll use the first one
	if len(resp.Results) == 0 {
		return &ModerationResponse{
			Flagged:    false,
			Categories: make(map[string]bool),
			Scores:     make(map[string]float64),
		}, nil
	}

	result := resp.Results[0]

	// Convert OpenAI categories to map format
	categories := map[string]bool{
		"harassment":             result.Categories.Harassment,
		"harassment/threatening": result.Categories.HarassmentThreatening,
		"hate":                   result.Categories.Hate,
		"hate/threatening":       result.Categories.HateThreatening,
		"self-harm":              result.Categories.SelfHarm,
		"self-harm/intent":       result.Categories.SelfHarmIntent,
		"self-harm/instructions": result.Categories.SelfHarmInstructions,
		"sexual":                 result.Categories.Sexual,
		"sexual/minors":          result.Categories.SexualMinors,
		"violence":               result.Categories.Violence,
		"violence/graphic":       result.Categories.ViolenceGraphic,
	}

	scores := map[string]float64{
		"harassment":             float64(result.CategoryScores.Harassment),
		"harassment/threatening": float64(result.CategoryScores.HarassmentThreatening),
		"hate":                   float64(result.CategoryScores.Hate),
		"hate/threatening":       float64(result.CategoryScores.HateThreatening),
		"self-harm":              float64(result.CategoryScores.SelfHarm),
		"self-harm/intent":       float64(result.CategoryScores.SelfHarmIntent),
		"self-harm/instructions": float64(result.CategoryScores.SelfHarmInstructions),
		"sexual":                 float64(result.CategoryScores.Sexual),
		"sexual/minors":          float64(result.CategoryScores.SexualMinors),
		"violence":               float64(result.CategoryScores.Violence),
		"violence/graphic":       float64(result.CategoryScores.ViolenceGraphic),
	}

	return &ModerationResponse{
		Flagged:    result.Flagged,
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
