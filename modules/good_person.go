package modules

import (
	"math/rand"
	"server-go/common"
	"strings"
)

func ReplaceBadWords(text string) string {
	
	for _, badVerb := range common.GoodPersonConfig.BadVerbs {
		// TOOD check if word has spaces
		ix := rand.Intn(len(common.GoodPersonConfig.ReplacementVerbs))
		text = strings.Replace(text, badVerb, common.GoodPersonConfig.ReplacementVerbs[ix], -1)
	}

	for _, badNoun := range common.GoodPersonConfig.BadNouns {
		ix := rand.Intn(len(common.GoodPersonConfig.ReplacementNouns))
		text = strings.Replace(text, badNoun, common.GoodPersonConfig.ReplacementNouns[ix], -1)
	}
	return text
}
