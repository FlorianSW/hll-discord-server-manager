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
)

type addTemplateData struct {
	Name string `discordgo:"name"`
}

type AddTemplateCommand struct {
	logger    *slog.Logger
	config    *internal.Config
	templates internal.Storage[resources.Template]
}

func NewAddTemplateCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Template]) *AddTemplateCommand {
	return &AddTemplateCommand{
		logger:    l,
		config:    c,
		templates: m,
	}
}

func (c *AddTemplateCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Adds a new template to be used when configuring a server",
		Options: []*discordgo.ApplicationCommandOption{{
			Name:        "name",
			Description: "A custom name for the template",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
			MinLength:   Int(1),
			MaxLength:   255,
		}},
	}
}

func (c *AddTemplateCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d addTemplateData
	if err := marshaller.Unmarshal(i.Interaction.ApplicationCommandData().Options, &d); err != nil {
		c.logger.Error("load-add-tpl-data", "error", err)
		ErrorResponse(s, i.Interaction, "Could not load data from interaction. Error: "+err.Error())
		return
	}

	tpl := resources.Template{
		TemplateId: uuid.NewString(),
		Name:       d.Name,
	}
	err := c.templates.Save(tpl)
	if err != nil {
		c.logger.Error("save-tpl", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error saving the template. Please try again. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String(fmt.Sprintf("The template with the name **%s** was added with ID %s.", tpl.Name, tpl.TemplateId)),
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
		return
	}
}
