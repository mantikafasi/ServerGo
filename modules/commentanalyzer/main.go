package commentanaylzer

import (
	"context"
	"server-go/common"

	"google.golang.org/api/commentanalyzer/v1alpha1"
	"google.golang.org/api/option"
)

var commentanalyzerService *commentanalyzer.Service

func init() {
	println("Initializing Comment Analyzer Service...")
	ctx := context.Background()
	var err error

	commentanalyzerService, err = commentanalyzer.NewService(ctx, option.WithAPIKey(common.Config.CommentAnalyzerAPIKey))

	if err != nil {
		panic(err)
	}
}

func AnalyzeComment(comment string) (*commentanalyzer.AnalyzeCommentResponse, error) {

	analyzecall := commentanalyzerService.Comments.Analyze(&commentanalyzer.AnalyzeCommentRequest{
		Comment: &commentanalyzer.TextEntry{
			Text: comment,
		},
		RequestedAttributes: map[string]commentanalyzer.AttributeParameters{
			"TOXICITY": {
				ScoreThreshold: 0.35,
			},
			"SEVERE_TOXICITY": {
				ScoreThreshold: 0.35,
			},
			"IDENTITY_ATTACK": {
				ScoreThreshold: 0.35,
			},
			"INSULT": {
				ScoreThreshold: 0.35,
			},
			"PROFANITY": {
				ScoreThreshold: 0.35,
			},
			"THREAT": {
				ScoreThreshold: 0.35,
			},
		},
	})

	analyzeResponse, err := analyzecall.Do()

	if err != nil {
		return nil, err
	}

	return analyzeResponse, nil
}

type AnalyzeResponse struct {
	AttributeName string
	AttributeScore float64
}

var SupportedLanguages = []string{
	"ar", "zh", "cs", "nl", "en,","fr", "de", "hi", "hi-Latn", "id", "it","ja", "ko", "pl", "pt", "ru", "es","sv",
}

func GetHighestScore(analyzeResponse *commentanalyzer.AnalyzeCommentResponse) (string, float64) {
	var highestScore float64
	var highestScoreName string

	for name, attribute := range analyzeResponse.AttributeScores {
		if attribute.SummaryScore.Value > highestScore {
			highestScore = attribute.SummaryScore.Value
			highestScoreName = name
		}
	}

	return highestScoreName, highestScore
}