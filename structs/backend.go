package structs

// Backender interface
type Backender interface {
	GetValue(rule Rule) (float64, error)
}

// Backend struct
type Backend struct {
	Name   string `mapstructure:"name"`
	Kind   string `mapstructure:"kind"`
	Region string `mapstructure:"region"`
	// Graphite-specific
	Host     string `mapstructure:"host"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
