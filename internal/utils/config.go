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
}

type awsConfig struct {
    InstanceID string `json:"instanceID"`
}

type discordConfig struct {
    BotToken string
    GuildID  string `json:"guildID"`
}

func MustLoadConfig(config *Config) {
    // load .env for credentials
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("error loading .env file %v", err)
    }
    config.BotToken = os.Getenv("BOT_TOKEN")

    file, err := os.Open("./config.json")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    if err := json.NewDecoder(file).Decode(&config); err != nil {
        log.Fatal(err)
    }
}
