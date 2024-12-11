package main

import (
	"log"
	"onemc/internal/aws"
	"onemc/internal/crafty"
	"onemc/internal/discord"
	"onemc/internal/utils"
	"time"
)

func init() {
	utils.MustLoadConfig(&config)
}

var (
	config utils.Config
)

func main() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			log.Println("Checking auto shutdown")
			stopQuery := crafty.StopQuery()
			if stopQuery == true {
				err := crafty.StopServer()
				if err != nil {
					log.Println(err)
				}
				aws.StopInstanceByID(config.InstanceID)
			}
		}
	}()
	// blocks thread
	discord.Run()
}
