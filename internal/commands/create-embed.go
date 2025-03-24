package commands

import (
	"github.com/bwmarrin/discordgo"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
	"strings"
)

type CreateEmbedCommand struct {
	logger  *slog.Logger
	config  *internal.Config
	servers internal.Storage[resources.Server]
}

const createEmbedPrefix = "create-embed"

func NewCreateEmbedCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Server]) *CreateEmbedCommand {
	return &CreateEmbedCommand{
		logger:  l,
		config:  c,
		servers: m,
	}
}

func (c *CreateEmbedCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Adds a message to this channel that allows discord users to manage registered servers.",
	}
}

func (c *CreateEmbedCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if c.config.EmbedMessage != nil {
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{{
				Title:       "Embed message already exists",
				Description: "There is already an embed registered. Do you want to recreate it?",
				Color:       ColorDarkRed,
			}},
			Components: &[]discordgo.MessageComponent{
				&discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						&discordgo.Button{
							Label:    "Yes, recreate",
							Style:    discordgo.DangerButton,
							CustomID: customId(createEmbedPrefix, "confirm-recreate"),
						},
					},
				},
			},
		})
		if err != nil {
			c.logger.Error("edit-response", "error", err)
		}
		return
	}
	m := c.createEmbed(s, i)
	if m == nil {
		return
	}
	c.config.EmbedMessage = &internal.EmbedMessage{
		ChannelId: m.ChannelID,
		MessageId: m.ID,
	}
	err := s.InteractionResponseDelete(i.Interaction)
	if err != nil {
		c.logger.Error("delete-response", "error", err)
	}
}

func (c *CreateEmbedCommand) createEmbed(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.Message {
	embeds, components, err := serversEmbed(c.servers)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return nil
	}
	m, err := s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds:     embeds,
		Components: components,
	})
	if err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not create a message with necessary message components. Error: "+err.Error())
		return nil
	}
	return m
}

func (c *CreateEmbedCommand) onConfirmRecreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if c.config.EmbedMessage != nil {
		err := s.ChannelMessageDelete(c.config.EmbedMessage.ChannelId, c.config.EmbedMessage.MessageId)
		if err != nil {
			c.logger.Error("delete-existing-embed", "error", err)
		}
	}

	m := c.createEmbed(s, i)

	if m == nil {
		return
	}
	c.config.EmbedMessage = &internal.EmbedMessage{
		ChannelId: m.ChannelID,
		MessageId: m.ID,
	}
	err := s.InteractionResponseDelete(i.Interaction)
	if err != nil {
		c.logger.Error("delete-response", "error", err)
	}
}

func (c *CreateEmbedCommand) onRefresh(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	embeds, components, err := serversEmbed(c.servers)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return
	}
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         i.Message.ID,
		Channel:    i.Message.ChannelID,
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not create a message with necessary message components. Error: "+err.Error())
		return
	}

	err = s.InteractionResponseDelete(i.Interaction)
	if err != nil {
		c.logger.Error("delete-response", "error", err)
	}
}

func (c *CreateEmbedCommand) OnMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.MessageComponentData().CustomID
	if id == customId(createEmbedPrefix, "confirm-recreate") {
		c.onConfirmRecreate(s, i)
	} else if id == customId(createEmbedPrefix, "refresh") {
		c.onRefresh(s, i)
	}
}

func (c *CreateEmbedCommand) CanHandle(customId string) bool {
	return strings.HasPrefix(customId, createEmbedPrefix)
}
