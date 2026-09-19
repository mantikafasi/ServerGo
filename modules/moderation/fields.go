package moderation

// ActionThreshold is the probability at which a field counts as a violation.
// Thresholds are product decisions, not model outputs: tune this on real
// reports before trusting it. Probabilities below it stay informational.
const ActionThreshold = 0.70

// field is one moderation dimension, phrased as a yes/no question so that a
// high probability always means "yes, this applies".
type field struct {
	Instructions string
	Yes          string
	No           string
}

// moderationFields is the moderation taxonomy: one narrow judgment per field,
// each scored independently in the same request. Group-specific fields stay
// separate from the general ones so callers can weight or gate them apart.
var moderationFields = map[string]field{
	"toxicity": {
		Instructions: "Is this text rude, hostile, or disrespectful toward anyone?",
		Yes:          "It shows hostility or disrespect.",
		No:           "It is civil, neutral, or friendly.",
	},
	"severe_toxicity": {
		Instructions: "Does this text contain extreme hatred or a call for violence or harm against people?",
		Yes:          "It expresses extreme hatred or calls for harm.",
		No:           "It stays well short of that.",
	},
	"identity_attack": {
		Instructions: "Does this text attack, demean, or express hostility toward a person or group because of an identity they hold?",
		Yes:          "It targets someone for their identity.",
		No:           "It does not target anyone for their identity.",
	},
	"insult": {
		Instructions: "Does this text insult, mock, or belittle a person or group?",
		Yes:          "It insults or belittles someone.",
		No:           "It does not insult or belittle anyone.",
	},
	"profanity": {
		Instructions: "Does this text use swear words or crude language?",
		Yes:          "It uses swear words or crude language.",
		No:           "It uses no swear words or crude language.",
	},
	"threat": {
		Instructions: "Does this text threaten to inflict harm on someone?",
		Yes:          "It threatens harm against a person or group.",
		No:           "It makes no threat of harm.",
	},
	"sexually_explicit": {
		Instructions: "Does this text explicitly describe sexual acts or sexual anatomy?",
		Yes:          "It describes sexual acts or anatomy explicitly.",
		No:           "It contains no explicit sexual description.",
	},
	"flirtation": {
		Instructions: "Does this text make sexual or romantic advances toward someone?",
		Yes:          "It makes sexual or romantic advances.",
		No:           "It makes no such advances.",
	},
	"sexuality": {
		Instructions: "Does this text attack or demean someone because of their sexual orientation?",
		Yes:          "It targets someone for their sexual orientation.",
		No:           "It does not target anyone for their sexual orientation.",
	},
	"gender": {
		Instructions: "Does this text attack or demean someone because of their gender or gender identity?",
		Yes:          "It targets someone for their gender or gender identity.",
		No:           "It does not target anyone for their gender.",
	},
	"religion": {
		Instructions: "Does this text attack or demean someone because of their religion or religious practice?",
		Yes:          "It targets someone for their religion or religious practice.",
		No:           "It does not target anyone for their religion.",
	},
	"race_ethnicity": {
		Instructions: "Does this text attack or demean someone because of their race or ethnicity?",
		Yes:          "It targets someone for their race or ethnicity.",
		No:           "It does not target anyone for their race or ethnicity.",
	},
	"disability": {
		Instructions: "Does this text attack or demean someone because of a disability or mental health condition?",
		Yes:          "It targets someone for a disability or mental health condition.",
		No:           "It does not target anyone for a disability or mental health condition.",
	},
	"age": {
		Instructions: "Does this text attack or demean someone because of their age?",
		Yes:          "It targets someone for their age.",
		No:           "It does not target anyone for their age.",
	},
	"self_harm": {
		Instructions: "Does this text promote, encourage, or describe an intent to self-harm or to die by suicide?",
		Yes:          "It promotes, encourages, or describes self-harm or suicide.",
		No:           "It contains no such content.",
	},
	"violence": {
		Instructions: "Does this text promote, glorify, or give instructions for violence or physical harm?",
		Yes:          "It promotes, glorifies, or instructs on violence.",
		No:           "It does none of those things.",
	},
	"spam": {
		Instructions: "Is this text spam, such as unsolicited advertising, a scam, or mass-repeated promotion?",
		Yes:          "It is advertising, a scam, or mass-repeated promotion.",
		No:           "It is ordinary user-written text.",
	},
}
