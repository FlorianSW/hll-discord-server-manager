package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
)

type deleteBroadcastMessageData struct {
	TemplateId   string `discordgo:"template"`
	MessageIndex int    `discordgo:"message"`
}

type autocompleteBroadcastMessageData struct {
	TemplateId string `discordgo:"template"`
}

type DeleteBroadcastMessageCommand struct {
	logger    *slog.Logger
	config    *internal.Config
	templates internal.Storage[resources.Template]
}

func NewDeleteBroadcastMessageCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Template]) *DeleteBroadcastMessageCommand {
	return &DeleteBroadcastMessageCommand{
		logger:    l,
		config:    c,
		templates: m,
	}
}

func (c *DeleteBroadcastMessageCommand) Definition(cmd string) *discordgo.ApplicationCommand {
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
			Name:         "message",
			Description:  "The message index to delete",
			Type:         discordgo.ApplicationCommandOptionInteger,
			Required:     true,
			Autocomplete: true,
		}},
	}
}

func (c *DeleteBroadcastMessageCommand) OnAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var typing string
	for _, option := range i.Interaction.ApplicationCommandData().Options {
		if option.Focused {
			typing = option.Name
		}
	}
	var choices []*discordgo.ApplicationCommandOptionChoice
	switch typing {
	case "template":
		choices = c.autocompleteTemplates(s, i)
	case "message":
		choices = c.autocompleteBroadcastMessages(s, i)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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

func (c *DeleteBroadcastMessageCommand) autocompleteTemplates(s *discordgo.Session, i *discordgo.InteractionCreate) (choices []*discordgo.ApplicationCommandOptionChoice) {
	l, err := c.templates.List()
	if err != nil {
		c.logger.Error("list-templates", "error", err)
		ErrorResponse(s, i.Interaction, "Could not list templates. Error: "+err.Error())
		return
	}
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
	return
}

func (c *DeleteBroadcastMessageCommand) autocompleteBroadcastMessages(s *discordgo.Session, i *discordgo.InteractionCreate) (choices []*discordgo.ApplicationCommandOptionChoice) {
	var d autocompleteBroadcastMessageData
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
	for idx, bm := range tpl.BroadcastMessage {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%d - %s", bm.Time, bm.Message),
			Value: idx,
		})
	}
	return
}

func (c *DeleteBroadcastMessageCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d deleteBroadcastMessageData
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
	var newBm []resources.BroadcastMessage
	for idx, bm := range tpl.BroadcastMessage {
		if idx != d.MessageIndex {
			newBm = append(newBm, bm)
		}
	}
	tpl.BroadcastMessage = newBm
	err = c.templates.Save(*tpl)
	if err != nil {
		c.logger.Error("save-tpl", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error saving the template. Please try again. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String("The message was deleted from the template."),
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
		return
	}
}
