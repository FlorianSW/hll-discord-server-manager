package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
	"net/url"
	"strings"
)

const credentialsPrefix = "credentials"

var (
	requiredPermissions = []string{
		"can_change_team_switch_cooldown",
		"can_view_team_switch_cooldown",
		"can_view_autobalance_threshold",
		"can_view_autobalance_enabled",
		"can_change_autobalance_enabled",
		"can_change_autobalance_threshold",
		"can_change_welcome_message",
		"can_view_welcome_message",
		"can_change_auto_broadcast_config",
		"can_view_auto_broadcast_config",
		"can_view_admins",
		"can_add_admin_roles",
		"can_remove_admin_roles",
		"can_unban_profanities",
		"can_view_profanities",
		"can_ban_profanities",
		"can_change_profanities",
		"can_view_playerids",
	}
)

type credentialsData struct {
	ServerId string `discordgo:"server"`
}

type CredentialsCommand struct {
	logger  *slog.Logger
	config  *internal.Config
	servers internal.Storage[resources.Server]
}

func NewCredentialsCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Server]) *CredentialsCommand {
	return &CredentialsCommand{
		logger:  l,
		config:  c,
		servers: m,
	}
}

func (c *CredentialsCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Manage credentials of a server",
		Options: []*discordgo.ApplicationCommandOption{{
			Name:         "server",
			Description:  "The server ID of which to manage credentials",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		}},
	}
}

func (c *CredentialsCommand) OnAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	l, err := c.servers.List()
	if err != nil {
		c.logger.Error("list-servers", "error", err)
		ErrorResponse(s, i.Interaction, "Could not list servers. Error: "+err.Error())
		return
	}
	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, id := range l {
		server, err := c.servers.Find(id)
		if err != nil {
			c.logger.Error("find-server", "error", err)
			ErrorResponse(s, i.Interaction, "Could not find server with ID "+id+". Error: "+err.Error())
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  server.Name,
			Value: server.ServerId,
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

func (c *CredentialsCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d credentialsData
	if err := marshaller.Unmarshal(i.Interaction.ApplicationCommandData().Options, &d); err != nil {
		c.logger.Error("load-credentials-data", "error", err)
		ErrorResponse(s, i.Interaction, "Could not load data from interaction. Error: "+err.Error())
		return
	}

	server, err := c.servers.Find(d.ServerId)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching server details. Error: "+err.Error())
		return
	}
	embeds, components := serverCredentialsEmbed(server)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		c.logger.Error("send-response", "error", err)
		return
	}
}

func serverCredentialsEmbed(s *resources.Server) (embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	tcadmin := "not set"
	if s.TCAdminCredentials != nil {
		tcadmin = fmt.Sprintf("%s (Service ID: %s)", s.TCAdminCredentials.BaseUrl, s.TCAdminCredentials.ServiceId)
	}
	crcon := "not set"
	if s.CRConCredentials != nil {
		crcon = s.CRConCredentials.BaseUrl
	}
	embeds = append(embeds, &discordgo.MessageEmbed{
		Color: ColorDarkGrey,
		Title: s.Name,
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "ID",
			Value: s.ServerId,
		}, {
			Name:   "CRCon Credentials",
			Value:  crcon,
			Inline: true,
		}, {
			Name:   "TCAdmin Credentials",
			Value:  tcadmin,
			Inline: true,
		}},
	})
	components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Set CRCon",
			CustomID: customId(credentialsPrefix, "set-crcon", s.ServerId),
			Style:    discordgo.PrimaryButton,
		},
		discordgo.Button{
			Label:    "Set TCAdmin",
			CustomID: customId(credentialsPrefix, "set-tcadmin", s.ServerId),
			Style:    discordgo.PrimaryButton,
		},
		discordgo.Button{
			Emoji:    &discordgo.ComponentEmoji{ID: "1283790096461594655"},
			CustomID: customId(credentialsPrefix, "refresh", s.ServerId),
			Style:    discordgo.SecondaryButton,
		},
	}})
	return
}

func (c *CredentialsCommand) onSetCredentialsRefreshClick(s *discordgo.Session, i *discordgo.InteractionCreate, serverId string) {
	server, err := c.servers.Find(serverId)
	if err != nil {
		c.logger.Error("find-server", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching server details. Error: "+err.Error())
		return
	}
	embeds, components := serverCredentialsEmbed(server)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
		},
	})
	if err != nil {
		c.logger.Error("edit-response", "error", err)
		return
	}
}

func (c *CredentialsCommand) OnMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.MessageComponentData().CustomID
	peek, _ := peekId(id)
	if matchesId(id, customId(credentialsPrefix, "set-crcon")) {
		c.onSetCredentialsClick(s, i, crconForm, peek)
	} else if matchesId(id, customId(credentialsPrefix, "set-tcadmin")) {
		c.onSetCredentialsClick(s, i, tcadminForm, peek)
	} else if matchesId(id, customId(credentialsPrefix, "refresh")) {
		c.onSetCredentialsRefreshClick(s, i, peek)
	}
}

type setCredentialsForm struct {
	Components []discordgo.MessageComponent
	Title      string
	CustomID   string
}

type setCRConFormData struct {
	Url    string `discordgo:"crcon_url"`
	ApiKey string `discordgo:"api_key"`
}

type setTCAdminFormData struct {
	Url       string `discordgo:"base_url"`
	ServiceId string `discordgo:"service_id"`
	Username  string `discordgo:"username"`
	Password  string `discordgo:"password"`
}

var (
	crconForm = setCredentialsForm{
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "crcon_url",
						Label:    "Community RCon URL",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "api_key",
						Label:    "API Key",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
		},
		Title:    "Set CRCon Credentials",
		CustomID: customId(credentialsPrefix, "confirm-crcon"),
	}
	tcadminForm = setCredentialsForm{
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "base_url",
						Label:    "Base URL without protocol",
						Value:    "qp.qonzer.com",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "service_id",
						Label:    "Service ID",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "username",
						Label:    "Username",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "password",
						Label:    "Password",
						Style:    discordgo.TextInputShort,
						Required: true,
					},
				},
			},
		},
		Title:    "Set TCAdmin Credentials",
		CustomID: customId(credentialsPrefix, "confirm-tcadmin"),
	}
)

func (c *CredentialsCommand) onSetCredentialsClick(s *discordgo.Session, i *discordgo.InteractionCreate, form setCredentialsForm, serverId string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:      form.Title,
			Components: form.Components,
			CustomID:   customId(form.CustomID, serverId),
		},
	})
	if err != nil {
		c.logger.Error("message-component-respond", "error", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
	}
}

func (c *CredentialsCommand) OnModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.ModalSubmitData().CustomID
	peek, _ := peekId(id)
	if matchesId(id, customId(credentialsPrefix, "confirm-crcon")) {
		c.onConfirmCRConCredentials(s, i, peek)
	} else if matchesId(id, customId(credentialsPrefix, "confirm-tcadmin")) {
		c.onConfirmTCAdminCredentials(s, i, peek)
	}
}

func (c *CredentialsCommand) onConfirmCRConCredentials(s *discordgo.Session, i *discordgo.InteractionCreate, serverId string) {
	var d setCRConFormData
	if err := marshaller.Unmarshal(i.ModalSubmitData().Components, &d); err != nil {
		c.logger.Error("parse-data", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
		return
	}
	u, err := url.Parse(d.Url)
	if err != nil {
		c.logger.Error("parse-url", "error", err)
		ErrorResponse(s, i.Interaction, "Could not verify permissions of the provided credentials. Error: "+err.Error())
		return
	}
	u.RawPath = ""
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	server, err := c.servers.Find(serverId)
	if err != nil {
		c.logger.Error("get-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+serverId+": "+err.Error())
		return
	}
	if server == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+serverId)
		return
	}

	creds := resources.CRConCredentials{
		BaseUrl: d.Url,
		ApiKey:  d.ApiKey,
	}
	client := crconClient(creds)
	if p, err := client.OwnPermissions(context.Background()); err != nil {
		c.logger.Error("request-permissions", "error", err)
		ErrorResponse(s, i.Interaction, "Could not verify permissions of the provided credentials. Error: "+err.Error())
		return
	} else if !p.Permissions.ContainsOnly(requiredPermissions) {
		ErrorResponse(
			s, i.Interaction,
			fmt.Sprintf(
				"The provided API key grants more or less permissions than the required ones. Please only provide the required permissions.\n\nProvided:\n```\n%s```\n\nRequired:\n```\n%s```",
				strings.Join(p.Permissions, "\n"),
				strings.Join(requiredPermissions, "\n"),
			),
		)
		return
	}

	server.CRConCredentials = &creds
	if err := c.servers.Save(*server); err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Couldn't save server data. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String("CRCon credentials set. Refresh the embed to see the new status."),
	})
	if err != nil {
		c.logger.Error("edit-original-message", "error", err)
	}
}

func (c *CredentialsCommand) onConfirmTCAdminCredentials(s *discordgo.Session, i *discordgo.InteractionCreate, serverId string) {
	var d setTCAdminFormData
	if err := marshaller.Unmarshal(i.ModalSubmitData().Components, &d); err != nil {
		c.logger.Error("parse-data", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
		return
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	server, err := c.servers.Find(serverId)
	if err != nil {
		c.logger.Error("get-server", "error", err)
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+serverId+": "+err.Error())
		return
	}
	if server == nil {
		ErrorResponse(s, i.Interaction, "Could not find server with ID "+serverId)
		return
	}

	creds := resources.TCAdminCredentials{
		BaseUrl:   d.Url,
		ServiceId: d.ServiceId,
		Username:  d.Username,
		Password:  d.Password,
	}
	client := tcadminClient(creds)
	if _, err := client.ServerInfo(d.ServiceId); err != nil {
		c.logger.Error("request-status", "error", err)
		ErrorResponse(s, i.Interaction, "Could not verify permissions of the provided credentials. Error: "+err.Error())
		return
	}

	server.TCAdminCredentials = &creds
	if err := c.servers.Save(*server); err != nil {
		c.logger.Error("save-server", "error", err)
		ErrorResponse(s, i.Interaction, "Couldn't save server data. Error: "+err.Error())
		return
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: String("TCAdmin credentials set. Refresh the embed to see the new status."),
	})
	if err != nil {
		c.logger.Error("edit-original-message", "error", err)
	}
}

func (c *CredentialsCommand) CanHandle(customId string) bool {
	return strings.HasPrefix(customId, credentialsPrefix)
}
