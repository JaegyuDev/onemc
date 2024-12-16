package main

import (
	"onemc/internal/crafty"
	"onemc/internal/discord"
	"onemc/internal/utils"
)

func init() {
	utils.MustLoadConfig(&config)
}

var (
	config utils.Config
)

func main() {
	go crafty.AutoShutdown(config.InstanceID)
	// blocks thread
	discord.Run()
}
