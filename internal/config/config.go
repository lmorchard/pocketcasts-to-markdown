package config

// Config holds application configuration
type Config struct {
	// Core settings
	Database string
	Verbose  bool
	Debug    bool
	LogJSON  bool

	// Pocket Casts credentials
	Email    string
	Password string
}
