package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
)

type addBroadcastMessageData struct {
	TemplateId string `discordgo:"template"`
	Time       int    `discordgo:"time"`
	Message    string `discordgo:"message"`
}

type AddBroadcastMessageCommand struct {
	logger    *slog.Logger
	config    *internal.Config
	templates internal.Storage[resources.Template]
}

func NewAddBroadcastMessageCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Template]) *AddBroadcastMessageCommand {
	return &AddBroadcastMessageCommand{
		logger:    l,
		config:    c,
		templates: m,
	}
}

func (c *AddBroadcastMessageCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Adds a broadcast message to a template",
		Options: []*discordgo.ApplicationCommandOption{{
			Name:         "template",
			Description:  "The ID of the template where to add the message",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		}, {
			Name:        "time",
			Description: "The time after the last message when this message should be send",
			Type:        discordgo.ApplicationCommandOptionInteger,
			Required:    true,
			MinLength:   Int(1),
			MaxLength:   3,
		}, {
			Name:        "message",
			Description: "The message to send",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
			MinLength:   Int(1),
			MaxLength:   255,
		}},
	}
}

func (c *AddBroadcastMessageCommand) OnAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	l, err := c.templates.List()
	if err != nil {
		c.logger.Error("list-templates", "error", err)
		ErrorResponse(s, i.Interaction, "Could not list templates. Error: "+err.Error())
		return
	}
	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, tid := range l {
		tpl, err := c.templates.Find(tid)
		if err != nil {
			c.logger.Error("find-templates", "error", err)
			ErrorResponse(s, i.Interaction, "Could not find template with ID "+tid+". Error: "+err.Error())
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  tpl.Name,
			Value: tpl.TemplateId,
		})
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		c.logger.Error("response", "error", err)
		ErrorResponse(s, i.Interaction, "Could not send response. Error: "+err.Error())
		return
	}
}

func (c *AddBroadcastMessageCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d addBroadcastMessageData
	if err := marshaller.Unmarshal(i.Interaction.ApplicationCommandData().Options, &d); err != nil {
		c.logger.Error("load-add-tpl-data", "error", err)
		ErrorResponse(s, i.Interaction, "Could not load data from interaction. Error: "+err.Error())
		return
	}
	tpl, err := c.templates.Find(d.TemplateId)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching template details. Error: "+err.Error())
		return
	}
	if tpl == nil {
		ErrorResponse(s, i.Interaction, "Could not find template with ID "+d.TemplateId)
		return
	}
	tpl.BroadcastMessage = append(tpl.BroadcastMessage, resources.BroadcastMessage{
		Time:    d.Time,
		Message: d.Message,
	})
	err = c.templates.Save(*tpl)
	if err != nil {
		c.logger.Error("save-tpl", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error saving the template. Please try again. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String("The message was added to the template."),
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
		return
	}
}
