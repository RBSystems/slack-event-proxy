package main

type SlackHelp struct {
	Building   string `json:"building"`
	Room       string `json:"room"`
	Text       string `json:"text"`
	Notes      string `json:"notes"`
	CallbackID string `json:callback_id"`
	TechName   string `json:techName"`
}

type UserDialog struct {
	TriggerID string `json:"trigger_id"`
	Dialog    Dialog `json:"dialog"`
}

type Dialog struct {
	TriggerID   string    `json:"trigger_id"`
	Title       string    `json:"title"`
	SubmitLabel string    `json:"submit_label"`
	Elements    []Element `json:"elements"`
	CallbackID  int64     `json:"callback_id"`
}

type Element struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Name  string `json:"name"`
}

type HelpData struct {
	Building string `json:"building"`
	Room     string `json:"room"`
	Notes    string
	TechName string
}
