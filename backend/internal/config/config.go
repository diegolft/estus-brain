// Package config loads runtime configuration from the environment. Access is
// gated by an in-app email/password login, which replaced the mutual-TLS edge
// that used to front the deployment: the certificate protected everything
// indiscriminately and could not tell the app who was calling, which the
// vault's reveal gate needs to know.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	// Where uploaded documents are written. Relative paths resolve against
	// the working directory the binary runs from, which in development is
	// backend/ — so the archive sits inside the project by default.
	DocumentsDir string
	// AssistantDir holds the assistant's local state: the MCP token file and
	// the scratch directories the CLI engines run in.
	AssistantDir string
	// MCPToken, when set, is the bearer token MCP clients must present;
	// otherwise one is generated once and kept in AssistantDir.
	MCPToken string
	// Timezone is the owner's: the assistant reads "today" and bare times in it.
	Timezone string
	// MultiUser switches off the engines that run on the owner's own
	// Claude/ChatGPT subscription login — those are for personal use only.
	MultiUser bool
	// TelegramBotToken is used when no token was saved on the settings screen.
	TelegramBotToken string
	// AdminEmail and AdminPassword seed the first account on a fresh
	// database. They are read on every boot but only ever act when the users
	// table is empty, so leaving them set on a running install is inert
	// rather than a standing back door.
	AdminEmail    string
	AdminPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DocumentsDir:     getEnv("DOCUMENTS_DIR", "data/documents"),
		AssistantDir:     getEnv("ASSISTANT_DIR", "data/assistant"),
		MCPToken:         os.Getenv("MCP_TOKEN"),
		Timezone:         getEnv("ASSISTANT_TZ", "America/Sao_Paulo"),
		MultiUser:        os.Getenv("ASSISTANT_MULTI_USER") == "true",
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		AdminEmail:       os.Getenv("ESTUS_ADMIN_EMAIL"),
		AdminPassword:    os.Getenv("ESTUS_ADMIN_PASSWORD"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
