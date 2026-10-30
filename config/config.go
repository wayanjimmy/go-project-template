package config

import (
	"fmt"
	"net/url"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Config struct {
	DatabaseURL   string `validate:"required"`
	RedisAddr     string `validate:"omitempty"`
	ServerPort    string `validate:"omitempty"`
	WebhookURL    string `validate:"omitempty"`
	WebhookSecret string `validate:"required_with=WebhookURL"`

	SecretType  string `validate:"required_with=SecretRoot SecretEncryptionParent SecretEncryptionName,omitempty,oneof=FILESYSTEM"`
	SecretRoot  string `validate:"required_with=SecretType SecretEncryptionParent SecretEncryptionName,omitempty"`
	SecretKeyID string `validate:"-"`

	SecretEncryptionParent string `validate:"required_with=SecretType SecretRoot SecretEncryptionName,omitempty"`
	SecretEncryptionName   string `validate:"required_with=SecretType SecretRoot SecretEncryptionParent,omitempty"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is required")
	}

	if err := validate.Struct(c); err != nil {
		return err
	}
	if c.WebhookURL != "" {
		u, err := url.Parse(c.WebhookURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return fmt.Errorf("WEBHOOK_URL must be an HTTP(S) URL without credentials")
		}
		if c.RedisAddr == "" {
			return fmt.Errorf("REDIS_ADDR is required for webhooks")
		}
	}

	if c.SecretType != "" {
		c.SecretKeyID = c.SecretEncryptionParent + "/" + c.SecretEncryptionName
	}

	return nil
}

func (c *Config) DatabaseConfig() *Config {
	return c
}

func (c *Config) RedisConfig() *Config {
	return c
}

func (c *Config) KeyManagerConfig() *Config {
	return c
}
