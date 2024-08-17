package config

type BaseLogsConfig struct {
	Level      string `yaml:"level"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
}

type WriteAheadLogging struct {
	BaseLogsConfig `yaml:",inline"`
	Enabled        bool `yaml:"enabled"`
}

type Config struct {
	Logs struct {
		BaseLogsConfig `yaml:",inline"`
		Filename       string `yaml:"filename"`
	} `yaml:"logs"`

	WriteAheadLogging WriteAheadLogging `yaml:"write_ahead_logging"`

	Printer struct {
		Name            string `yaml:"name"`
		MonitorInterval int    `yaml:"monitor_interval_ms"`
	} `yaml:"printer"`

	Engine struct {
		Handlers             []HandlerConfig `yaml:"handlers"`
		IgnoreRecoveryErrors bool            `yaml:"ignore_recovery_errors"`
		MaxWorkers           int             `yaml:"max_workers"`
	} `yaml:"engine"`
	Workdir string `yaml:"workdir"`
}

type HandlerConfig struct {
	Name   string                 `yaml:"name"`
	Retry  HandlerRetryMechanism  `yaml:"retry,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

type HandlerRetryMechanism struct {
	MaxRetries      int `yaml:"max_retries"`
	BackOffInterval int `yaml:"backoff_interval"`
}
