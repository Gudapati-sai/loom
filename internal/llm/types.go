package llm

// Option is one selectable choice presented to the user, always paired
// with a short explanation so the choice is informed, not blind.
type Option struct {
	Label       string `json:"label"`
	Explanation string `json:"explanation"`
}

// Question is the structured shape every question in the wizard takes,
// whether it came from the static fallback or a local model. Keeping this
// strict is what lets a small local model's output be validated instead
// of trusted on faith.
type Question struct {
	Prompt      string   `json:"question"`
	Options     []Option `json:"options"`
	AllowCustom bool     `json:"allow_custom"`
	FreeText    bool     `json:"free_text"`
}

func (q Question) Valid() bool {
	if q.Prompt == "" {
		return false
	}
	if q.FreeText {
		return true
	}
	return len(q.Options) > 0
}
