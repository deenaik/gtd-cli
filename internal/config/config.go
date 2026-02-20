package config

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

var (
	cfg  *Config
	once sync.Once
)

type Config struct {
	DataDir      string `yaml:"data_dir"`
	DBPath       string `yaml:"db_path"`
	OpenAIKey    string `yaml:"-"`
	OpenAIModel  string `yaml:"openai_model"`
	TokenBudget  int    `yaml:"token_budget"`
	GogPath      string `yaml:"gog_path"`
	UserEmail    string `yaml:"user_email"`
	AgentLogPath string `yaml:"agent_log_path"`

	// Agent intervals (seconds)
	EmailInterval    int `yaml:"email_interval"`
	CalendarInterval int `yaml:"calendar_interval"`
	DeadlineInterval int `yaml:"deadline_interval"`
	ProgressInterval int `yaml:"progress_interval"`

	// Notifications
	ChatNotify bool `yaml:"chat_notify"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".gtd")
	return &Config{
		DataDir:          dataDir,
		DBPath:           filepath.Join(dataDir, "gtd.db"),
		OpenAIModel:      "gpt-4o-mini",
		TokenBudget:      100000,
		GogPath:          "gog",
		UserEmail:        "deepak@digite.com",
		AgentLogPath:     filepath.Join(dataDir, "agent.log"),
		EmailInterval:    300,
		CalendarInterval: 60,
		DeadlineInterval: 900,
		ProgressInterval: 3600,
		ChatNotify:       false,
	}
}

func DataDir() string {
	return Get().DataDir
}

func Get() *Config {
	once.Do(func() {
		cfg = load()
	})
	return cfg
}

func load() *Config {
	c := DefaultConfig()

	// Load .env from current dir (best effort)
	_ = godotenv.Load()

	// Load config file
	configPath := filepath.Join(c.DataDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = yaml.Unmarshal(data, c)
	}
	// Recompute derived paths if data_dir changed
	if c.DBPath == "" {
		c.DBPath = filepath.Join(c.DataDir, "gtd.db")
	}
	if c.AgentLogPath == "" {
		c.AgentLogPath = filepath.Join(c.DataDir, "agent.log")
	}

	// Env overrides
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		c.OpenAIKey = v
	}
	if v := os.Getenv("GTD_DATA_DIR"); v != "" {
		c.DataDir = v
		c.DBPath = filepath.Join(v, "gtd.db")
		c.AgentLogPath = filepath.Join(v, "agent.log")
	}
	if v := os.Getenv("GTD_GOG_PATH"); v != "" {
		c.GogPath = v
	}
	if v := os.Getenv("GTD_TOKEN_BUDGET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.TokenBudget = n
		}
	}

	return c
}

func EnsureDataDir() error {
	return os.MkdirAll(Get().DataDir, 0o755)
}

func WriteDefault() error {
	if err := EnsureDataDir(); err != nil {
		return err
	}
	c := DefaultConfig()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.DataDir, "config.yaml"), data, 0o644)
}
