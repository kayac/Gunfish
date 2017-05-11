package gunfish

import "encoding/json"

type Result interface {
	Err() error
	Status() int
	Provider() string
	RecipientIdentifier() string
	ExtraKeys() []string
	ExtraValue(string) string
	json.Marshaler
}
