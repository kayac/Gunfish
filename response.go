package gunfish

type Result interface {
	Err() error
	Status() int
	Provider() string
	RecipientIdentifier() string
	ExtraKeys() []string
	ExtraValue(string) string
}
