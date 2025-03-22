package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"github.com/google/uuid"
	"log/slog"
	"strings"
)

type addServerData struct {
	Name string `discordgo:"name"`
}

type AddServerCommand struct {
	logger  *slog.Logger
	config  *internal.Config
	servers internal.Servers
}

func NewAddServerCommand(l *slog.Logger, c *internal.Config, m internal.Servers) *AddServerCommand {
	return &AddServerCommand{
		logger:  l,
		config:  c,
		servers: m,
	}
}

func (c *AddServerCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Adds a new server to the list of managed servers.",
		Options: []*discordgo.ApplicationCommandOption{{
			Name:        "name",
			Description: "A custom name for the server. This is not the server name as it appears in the in-game server list",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
			MinLength:   Int(1),
			MaxLength:   255,
		}},
	}
}

func (c *AddServerCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d addServerData
	if err := marshaller.Unmarshal(i.Interaction.ApplicationCommandData().Options, &d); err != nil {
		c.logger.Error("load-add-server-data", "error", err)
		ErrorResponse(s, i.Interaction, "Could not load data from interaction. Error: "+err.Error())
		return
	}

	server := resources.Server{
		ServerId: uuid.NewString(),
		Name:     d.Name,
	}
	err := c.servers.Save(server)
	if err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error saving the server. Please try again. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String(fmt.Sprintf("The server with the name **%s** was added with ID %s.", server.Name, server.ServerId)),
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
		return
	}
}

func (c *AddServerCommand) CanHandle(customId string) bool {
	return strings.HasPrefix(customId, "add-server")
}
