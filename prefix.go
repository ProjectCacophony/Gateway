package main

// Stores the Prefix Config for a Guild
type PrefixEntry struct {
	GuildID string
	Prefix  []string
}

func GetPrefixes() (prefixes []PrefixEntry, err error) {
	return []PrefixEntry{ // example prefix config
		{
			GuildID: "435420687906111498",
			Prefix:  []string{"!"},
		},
		{
			GuildID: "199845954273280000",
			Prefix:  []string{"_", "/"},
		},
		{
			GuildID: "339227598544568340",
			Prefix:  []string{"_"},
		},
	}, nil
}
