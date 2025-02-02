// TODO: Send message on autostop initiate, cancel and success

package discord

import (
	"flag"
	"github.com/bwmarrin/discordgo"
	"log"
	"onemc/internal/aws"
	"onemc/internal/crafty"
	"onemc/internal/utils"
	"os"
	"os/signal"
	"time"
)

var (
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var (
	config utils.Config
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	utils.MustLoadConfig(&config)
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	integerOptionMinValue          = 1.0
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionAll

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "server",
			Description: "managing the status of the minecraft server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "start the minecraft server instance",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stop",
					Description: "stop the minecraft server instance",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "status",
					Description: "get the minecraft server status",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"server": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			content := ""

			switch options[0].Name {
			case "start":
				content = "Starting server..."
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})
				if err != nil {
					log.Printf("Failed to respond to interaction: %v\n", err)
					return
				}

				aws.StartAWSInstanceByID(config.InstanceID)
				_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Server successfully started",
				})

				err = crafty.StartMCServer()
				if err != nil {
					log.Printf("Failed to start instance, may not be online?: %v\n", err)
				}

				if err != nil {
					log.Printf("Failed to send follow-up message: %v\n", err)
				}
				for crafty.CheckRunning() == false {
					time.Sleep(5 * time.Second)
				}

			case "stop":
				err := crafty.StopMCServer()
				if err != nil {
					log.Printf("Failed to stop server: %v", err)
					return
				}

				content = "Stopping server..."

				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})
				if err != nil {
					log.Printf("Failed to respond to interaction: %v", err)
					return
				}
				time.Sleep(10 * time.Second)
				aws.StopAWSInstanceByID(config.InstanceID)
				// Send follow-up message
				_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Server successfully stopped",
				})
				if err != nil {
					log.Printf("Failed to send follow-up message: %v", err)
				}

				// TODO: Make status command work
			//case "status":
			//	crafty.UpdateStats()
			//	switch *crafty.Running {
			//	case true:
			//		status = "Running"
			//	case false:
			//		status = "Not running"
			//	}
			//
			//	content = fmt.Sprintf("The server is currently %s", status)
			//	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			//		Type: discordgo.InteractionResponseChannelMessageWithSource,
			//		Data: &discordgo.InteractionResponseData{
			//			Content: content,
			//		},
			//	})
			//	if err != nil {
			//		log.Printf("Failed to respond to interaction: %v", err)
			//		return
			//	}

			default:
				content = "Unknown subcommand. Please use `start` or `stop`."
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})
				if err != nil {
					log.Printf("Failed to respond to interaction: %v", err)
				}
			}
		},
	}
)

func Run() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, config.GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")

		for _, v := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, config.GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("Gracefully shutting down.")
}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}
