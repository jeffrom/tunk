package config

func GetDefault() Config {
	return Config{
		Policies: []string{"conventional-lax", "lax"},
		Branches: []string{"main", "master"},
	}
}
