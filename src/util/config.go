package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type JWTConfig struct {
	Secret     string
	Expiration int
}

type ConfigReader struct {
	Db        DB
	Server    Server
	JWTConfig JWTConfig
	Admin     Admin
}

func NewConfigReader() *ConfigReader {
	return &ConfigReader{Db: DB{}, Server: Server{}, JWTConfig: JWTConfig{}, Admin: Admin{}}
}

func (c *ConfigReader) Setup() {
	config := os.Getenv("CONFIG_YAML_AUTH")
	// config = "a"

	configPath := filepath.Join("configs", "config.yaml")

	fmt.Println("Making the directory for config")
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		log.Fatal(err)
	}

	// err = os.WriteFile(configPath, []byte(config), 0600)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if config != "" {
		fmt.Println("CONFIG_YAML environment variable is set. Writing to config.yaml.")
		err := os.WriteFile(configPath, []byte(config), 0600)
		if err != nil {
			fmt.Println("Error writing file.")
			panic(err)
		}
		fmt.Println("Wrote to config")
	} else {
		fmt.Println("CONFIG_YAML environment variable is not set. Using existing config.yaml.")
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs") // current directory

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	fmt.Println("config read successfully")

	// -----------------------
	// Extract values
	// -----------------------
	c.Db.Host = viper.GetString("database.host")
	c.Db.Port = viper.GetInt("database.port")
	c.Db.User = viper.GetString("database.user")
	c.Db.Password = viper.GetString("database.password")
	c.Db.Dbname = viper.GetString("database.dbname")
	c.Db.Sslmode = viper.GetString("database.sslmode")
	c.Db.Timezone = viper.GetString("database.timezone")

	c.Server.Address = viper.GetString("server.address")
	c.Server.Port = viper.GetString("server.port")

	c.JWTConfig.Secret = viper.GetString("jwt.secret")
	c.JWTConfig.Expiration = viper.GetInt("jwt.expiration")

	c.Admin.Email = viper.GetString("admin.email")
	c.Admin.Password = viper.GetString("admin.password")
}

type DB struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Sslmode  string
	Timezone string
}

func (d *DB) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		d.Host, d.User, d.Password, d.Dbname, d.Port, d.Sslmode, d.Timezone,
	)
}

type Admin struct {
	Email    string
	Password string
}

type Server struct {
	Address string
	Port    string
}

func (s *Server) ConstructUrl() string {
	return s.Address + ":" + s.Port
}
