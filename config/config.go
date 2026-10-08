package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Security SecurityConfig `yaml:"security"`
}

type SecurityConfig struct {
	DefaultAdminPassword string `yaml:"default_admin_password"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	DBName    string `yaml:"dbname"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parseTime"`
	Loc       string `yaml:"loc"`
}

type AuthConfig struct {
	JWTSecret        string `yaml:"jwt_secret"`
	TokenExpireHours int    `yaml:"token_expire_hours"`
}

var (
	GlobalConfig *Config
	DB           *gorm.DB
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	GlobalConfig = &cfg
	return &cfg, nil
}

func InitDB() (*gorm.DB, error) {
	if GlobalConfig == nil {
		return nil, fmt.Errorf("global config not initialized")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		GlobalConfig.Database.User,
		GlobalConfig.Database.Password,
		GlobalConfig.Database.Host,
		GlobalConfig.Database.Port,
		GlobalConfig.Database.DBName,
		GlobalConfig.Database.Charset,
		GlobalConfig.Database.ParseTime,
		GlobalConfig.Database.Loc,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	DB = db
	return db, nil
}
