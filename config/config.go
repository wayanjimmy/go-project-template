package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Config struct {
	DatabaseURL string `validate:"required"`
	RedisAddr   string `validate:"omitempty"`
	ServerPort  string `validate:"omitempty"`

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
