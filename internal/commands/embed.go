package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-crcon"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
)

const embedPrefix = "embed"

type setNamePasswordData struct {
	Name     string `discordgo:"name"`
	Password string `discordgo:"password"`
}

type EmbedCommand struct {
	logger    *slog.Logger
	config    *internal.Config
	servers   internal.Storage[resources.Server]
	templates internal.Storage[resources.Template]
}

func NewEmbedCommand(l *slog.Logger, c *internal.Config, s internal.Storage[resources.Server], t internal.Storage[resources.Template]) *EmbedCommand {
	return &EmbedCommand{
		logger:    l,
		config:    c,
		servers:   s,
		templates: t,
	}
}

func (c *EmbedCommand) OnMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cid := i.Interaction.MessageComponentData().CustomID
	if cid == customId(embedPrefix, "select-server") {
		c.onSelectServer(s, i)
	} else if matchesId(cid, customId(embedPrefix, "refresh")) {
		peek, _ := peekId(cid)
		c.onRefresh(s, i, peek)
	} else if matchesId(cid, customId(embedPrefix, "set-name-password")) {
		peek, _ := peekId(cid)
		c.onSetNamePassword(s, i, peek)
	} else if matchesId(cid, customId(embedPrefix, "select-template")) {
		peek, _ := peekId(cid)
		c.onSelectTemplate(s, i, peek)
	} else if matchesId(cid, customId(embedPrefix, "save-restart")) {
		peek, _ := peekId(cid)
		c.onSaveRestart(s, i, peek)
	}
}

func (c *EmbedCommand) onSetNamePassword(s *discordgo.Session, i *discordgo.InteractionCreate, sid string) {
	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find server with ID "+sid+". Error: "+err.Error())
		return
	}
	if server == nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+". Error: "+err.Error())
		return
	}

	serverName := ""
	serverPassword := ""
	if server.PendingUpdate != nil {
		serverName = server.PendingUpdate.ServerName
		serverPassword = server.PendingUpdate.ServerPassword
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title: "Set Server Name and Password",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						Label:    "Name",
						Value:    serverName,
						CustomID: "name",
						Style:    discordgo.TextInputShort,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						Label:    "Password",
						Value:    serverPassword,
						CustomID: "password",
						Style:    discordgo.TextInputShort,
					},
				}},
			},
			CustomID: customId(embedPrefix, "confirm-name-password", sid),
		},
	})
	if err != nil {
		c.logger.Error("message-component-respond", "error", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
	}
}

func (c *EmbedCommand) onSelectServer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	sid := i.Interaction.MessageComponentData().Values[0]
	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find server with ID "+sid+". Error: "+err.Error())
		return
	}
	if server == nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+". Error: "+err.Error())
		return
	}

	if server.PendingUpdate != nil {
		server.PendingUpdate = nil
		err = c.servers.Save(*server)
		if err != nil {
			c.logger.Error("remove-pending-update", "error", err)
		}
	}

	embeds, components, err := serverEmbed(c.templates, *server)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
	}
}

func (c *EmbedCommand) onSelectTemplate(s *discordgo.Session, i *discordgo.InteractionCreate, sid string) {
	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find server with ID "+sid+". Error: "+err.Error())
		return
	}
	if server == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+". Error: "+err.Error())
		return
	}

	tplId := i.Interaction.MessageComponentData().Values[0]
	template, err := c.templates.Find(tplId)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find template with ID "+tplId+". Error: "+err.Error())
		return
	}
	if template == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+tplId+". Error: "+err.Error())
		return
	}
	if server.PendingUpdate == nil {
		server.PendingUpdate = &resources.ServerUpdate{}
	}
	server.PendingUpdate.TemplateId = tplId

	err = c.servers.Save(*server)
	if err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error saving server. Error: "+err.Error())
		return
	}

	embeds, components, err := serverEmbed(c.templates, *server)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
		},
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
	}
}

func (c *EmbedCommand) onSaveRestart(s *discordgo.Session, i *discordgo.InteractionCreate, sid string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find server with ID "+sid+". Error: "+err.Error())
		return
	}
	if server == nil || server.CRConCredentials == nil || server.TCAdminCredentials == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+".")
		return
	}
	if server.PendingUpdate == nil {
		ErrorResponse(s, i.Interaction, "There is no pending update for this server. Please start over by selecting the server again.")
		return
	}
	template, err := c.templates.Find(server.PendingUpdate.TemplateId)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find template with ID "+server.PendingUpdate.TemplateId+". Error: "+err.Error())
		return
	}
	if template == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+server.PendingUpdate.TemplateId+".")
		return
	}

	var errors []error
	ctx := context.Background()
	cc := crconClient(*server.CRConCredentials)
	config := crcon.AutoBroadcastConfig{Enabled: true, Randomize: false}
	for _, message := range template.BroadcastMessage {
		config.Messages = append(config.Messages, crcon.BroadcastMessage{
			TimeSec: message.Time,
			Message: message.Message,
		})
	}
	err = cc.SetAutoBroadcastConfig(ctx, config)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Auto-Broadcast: %w", err))
	}
	err = cc.SetWelcomeMessage(ctx, template.WelcomeMessage)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Welcome message: %w", err))
	}
	err = cc.SetTeamSwitchCooldown(ctx, template.TeamSwitchCooldown)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Team-Switch-Cooldown: %w", err))
	}
	err = cc.SetAutoBalanceThreshold(ctx, template.AutoBalanceThreshold)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Auto-Balance threshold: %w", err))
	}
	err = cc.SetProfanities(ctx, template.ProfanityFilter)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Profanities: %w", err))
	}

	tc := tcadminClient(*server.TCAdminCredentials)
	err = tc.SetServerInfo(server.TCAdminCredentials.ServiceId, server.PendingUpdate.ServerName, server.PendingUpdate.ServerPassword)
	if err != nil {
		errors = append(errors, fmt.Errorf("updating Server name and password: %w", err))
	}
	if err == nil && server.PendingUpdate.RequiresRestart() {
		_, err = tc.Restart(server.TCAdminCredentials.ServiceId)
		if err != nil {
			errors = append(errors, fmt.Errorf("restarting server: %w", err))
		}
	}

	server.PendingUpdate = nil
	err = c.servers.Save(*server)
	if err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error saving server. Error: "+err.Error())
		return
	}

	message := "The server was successfully prepared."
	if len(errors) != 0 {
		message = "Some settings could not be updated. Any not mentioned setting was made successfully. Errors:\n\n"
		for _, e := range errors {
			message += "* " + e.Error() + "\n"
		}
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &message,
		Embeds:     &[]*discordgo.MessageEmbed{},
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
	}
}

func (c *EmbedCommand) onRefresh(s *discordgo.Session, i *discordgo.InteractionCreate, sid string) {
	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Error trying to find server with ID "+sid+". Error: "+err.Error())
		return
	}
	if server == nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+". Error: "+err.Error())
		return
	}

	embeds, components, err := serverEmbed(c.templates, *server)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
		},
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
	}
}

func (c *EmbedCommand) OnModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.ModalSubmitData().CustomID
	peek, _ := peekId(id)
	if matchesId(id, customId(embedPrefix, "confirm-name-password")) {
		c.onConfirmNamePassword(s, i, peek)
	}
}

func (c *EmbedCommand) onConfirmNamePassword(s *discordgo.Session, i *discordgo.InteractionCreate, sid string) {
	var d setNamePasswordData
	if err := marshaller.Unmarshal(i.ModalSubmitData().Components, &d); err != nil {
		c.logger.Error("parse-data", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
		return
	}
	server, err := c.servers.Find(sid)
	if err != nil {
		c.logger.Error("get-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid+": "+err.Error())
		return
	}
	if server == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+sid)
		return
	}

	if server.PendingUpdate == nil {
		server.PendingUpdate = &resources.ServerUpdate{}
	}
	server.PendingUpdate.ServerName = d.Name
	server.PendingUpdate.ServerPassword = d.Password
	if err := c.servers.Save(*server); err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Couldn't save server data. Error: "+err.Error())
		return
	}
	embeds, components, err := serverEmbed(c.templates, *server)
	if err != nil {
		c.logger.Error("create-message-embeds", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error creating the message components. Error: "+err.Error())
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
		},
	})
	if err != nil {
		c.logger.Error("edit-original-message", "error", err)
	}
}

func (c *EmbedCommand) CanHandle(customId string) bool {
	return matchesId(customId, embedPrefix)
}
