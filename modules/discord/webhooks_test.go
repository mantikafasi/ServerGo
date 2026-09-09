package discord

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
)

func TestCannedAppealDenyReasons(t *testing.T) {
	expectedReasons := map[string]struct {
		Label string
		Text  string
	}{
		"idiot": {
			Label: "Idiot",
			Text:  "You wrote such a dumb reason even I could think of a better one",
		},
		"nsfw_allegations": {
			Label: "NSFW allegations",
			Text:  "We don't allow allegations of this nature because they are inflammatory, difficult to verify, and tend to lead to unnecessary drama. This isn't a determination of whether the claim is true or false; it's simply content we don't want on ReviewDB. If you have credible evidence of actual misconduct, please report it to Discord or, where applicable, the relevant authorities rather than using ReviewDB to make or debate the allegation.",
		},
		"hacked": {
			Label: "Hacked",
			Text:  "You are responsible for what happens to your account. In the future, ensure that only you has access to it",
		},
	}

	if len(CannedAppealDenyReasons) != len(expectedReasons) {
		t.Fatalf("expected %d canned reasons, got %d", len(expectedReasons), len(CannedAppealDenyReasons))
	}

	for _, canned := range CannedAppealDenyReasons {
		expected, exists := expectedReasons[canned.Value]
		if !exists {
			t.Errorf("unexpected canned reason value: %s", canned.Value)
			continue
		}
		if canned.Label != expected.Label {
			t.Errorf("for %s: expected label %q, got %q", canned.Value, expected.Label, canned.Label)
		}
		if canned.Text != expected.Text {
			t.Errorf("for %s: expected text %q, got %q", canned.Value, expected.Text, canned.Text)
		}

		gotText, ok := GetCannedAppealDenyReason(canned.Value)
		if !ok || gotText != expected.Text {
			t.Errorf("GetCannedAppealDenyReason(%q) = (%q, %t), want (%q, true)", canned.Value, gotText, ok, expected.Text)
		}
	}

	if _, ok := GetCannedAppealDenyReason("non_existent"); ok {
		t.Errorf("expected GetCannedAppealDenyReason with invalid key to return ok=false")
	}
}

func TestAppealWebhookComponentsSerialization(t *testing.T) {
	appealID := int32(42)
	options := make([]discordSelectOptionForTest, len(CannedAppealDenyReasons))
	for i, canned := range CannedAppealDenyReasons {
		options[i] = discordSelectOptionForTest{
			Label: canned.Label,
			Value: canned.Value,
		}
	}

	components := []WebhookComponent{
		{
			Type: 1,
			Components: []WebhookComponent{
				{
					Type:     2,
					Label:    "Accept",
					Style:    3,
					CustomID: fmt.Sprintf("accept_appeal:%d", appealID),
				},
				{
					Type:     2,
					Label:    "Deny",
					Style:    4,
					CustomID: fmt.Sprintf("text_deny_appeal:%d", appealID),
				},
			},
		},
		{
			Type: 1,
			Components: []WebhookComponent{
				{
					Type:        3,
					CustomID:    fmt.Sprintf("canned_deny_appeal:%d", appealID),
					Placeholder: "Deny with canned response...",
					Options: func() []discord.SelectOption {
						opts := make([]discord.SelectOption, len(CannedAppealDenyReasons))
						for idx, r := range CannedAppealDenyReasons {
							opts[idx] = discord.SelectOption{Label: r.Label, Value: r.Value}
						}
						return opts
					}(),
				},
			},
		},
	}

	data := WebhookData{
		Username:   "ReviewDB Appeals",
		Components: components,
	}

	marshaled, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(marshaled, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	comps, ok := parsed["components"].([]interface{})
	if !ok || len(comps) != 2 {
		t.Fatalf("expected 2 action rows in components, got %v", parsed["components"])
	}
}

type discordSelectOptionForTest struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
