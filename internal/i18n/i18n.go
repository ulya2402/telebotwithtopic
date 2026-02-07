package i18n

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Localizer struct {
	translations map[string]map[string]string
}

func NewLocalizer() *Localizer {
	loc := &Localizer{
		translations: make(map[string]map[string]string),
	}
	loc.loadLanguage("en")
	loc.loadLanguage("id")
	return loc
}

func (l *Localizer) loadLanguage(langCode string) {
	path := filepath.Join("locales", langCode+".json")
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening locale file %s: %v", path, err)
		return
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)
	var result map[string]string
	if err := json.Unmarshal(byteValue, &result); err != nil {
		log.Printf("Error unmarshalling locale %s: %v", langCode, err)
		return
	}
	l.translations[langCode] = result
	log.Printf("Loaded language: %s", langCode)
}

func (l *Localizer) Get(langCode, key string) string {
	if texts, ok := l.translations[langCode]; ok {
		if val, ok := texts[key]; ok {
			return val
		}
	}
	
	if texts, ok := l.translations["en"]; ok {
		if val, ok := texts[key]; ok {
			return val
		}
	}
	return key
}