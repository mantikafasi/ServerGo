package modules

import (
	"math/rand"
	"server-go/common"
	"strings"
)

func ReplaceBadWords(text string) string {

	words := strings.Split(text, " ")
	newText := ""

out:
	for _, word := range words {
		lower := " " + strings.ToLower(word)

		for _, badNoun := range common.GoodPersonConfig.BadNouns {
			ix := rand.Intn(len(common.GoodPersonConfig.ReplacementNouns))

			filtered := strings.Replace(lower, " "+badNoun, " "+common.GoodPersonConfig.ReplacementNouns[ix], -1)

			if strings.TrimSpace(filtered) != word {
				newText += strings.TrimSpace(filtered) + " "
				continue out
			}
		}

		for _, badVerb := range common.GoodPersonConfig.BadVerbs {
			ix := rand.Intn(len(common.GoodPersonConfig.ReplacementVerbs))

			filtered := strings.Replace(lower, " "+badVerb, " "+common.GoodPersonConfig.ReplacementVerbs[ix], -1)

			if strings.TrimSpace(filtered) != word {
				newText += strings.TrimSpace(filtered) + " "
				continue out
			}
		}

		newText += word + " "
	}
	return newText
}
