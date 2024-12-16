package utils

import (
	"encoding/json"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	awsConfig
	discordConfig
	craftyConfig
}

type awsConfig struct {
	InstanceID string `json:"instanceID"`
}

type discordConfig struct {
	BotToken string
	GuildID  string `json:"guildID"`
}

type craftyConfig struct {
	Username string
	Password string
	URL      string `json:"url"`
	ServerID string `json:"serverID"`
}

func MustLoadConfig(config *Config) {
	// load .env for credentials
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error loading .env file %v", err)
	}
	config.BotToken = os.Getenv("BOT_TOKEN")
	config.craftyConfig.Username = os.Getenv("CRAFTY_USER")
	config.craftyConfig.Password = os.Getenv("CRAFTY_PASS")

	file, err := os.Open("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatal(err)
	}

}
