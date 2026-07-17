package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed assets/english.json
var englishJSON []byte

type languageFile struct {
	Name  string   `json:"name"`
	Words []string `json:"words"`
}

func loadWords() ([]string, error) {
	var language languageFile
	if err := json.Unmarshal(englishJSON, &language); err != nil {
		return nil, fmt.Errorf("parse embedded word list: %w", err)
	}
	if len(language.Words) == 0 {
		return nil, fmt.Errorf("embedded word list is empty")
	}
	return language.Words, nil
}
