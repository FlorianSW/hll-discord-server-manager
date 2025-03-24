package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/marshaller"
	. "github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
	"strconv"
	"strings"
)

const templatesPrefix = "templates"

type templateData struct {
	Id string `discordgo:"template"`
}

type messagesData struct {
	WelcomeMessage     string `discordgo:"welcome-message"`
	ServerNameTemplate string `discordgo:"server-name-template"`
}

type profanityData struct {
	Filter string `discordgo:"profanity-filter"`
}

func (p profanityData) ProfanityFilter() []string {
	return strings.Split(p.Filter, "\n")
}

type thresholdsData struct {
	TeamSwitchCooldown   string `discordgo:"team-switch-cooldown"`
	AutoBalanceThreshold string `discordgo:"auto-balance-threshold"`
}

func (t thresholdsData) teamSwitchCooldown() int {
	if v, err := strconv.Atoi(t.TeamSwitchCooldown); err != nil {
		return 0
	} else {
		return v
	}
}

func (t thresholdsData) autoBalanceThreshold() int {
	if v, err := strconv.Atoi(t.AutoBalanceThreshold); err != nil {
		return 0
	} else {
		return v
	}
}

type TemplatesCommand struct {
	logger    *slog.Logger
	config    *internal.Config
	templates internal.Storage[resources.Template]
}

func NewTemplatesCommand(l *slog.Logger, c *internal.Config, m internal.Storage[resources.Template]) *TemplatesCommand {
	return &TemplatesCommand{
		logger:    l,
		config:    c,
		templates: m,
	}
}

func (c *TemplatesCommand) Definition(cmd string) *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        cmd,
		Description: "Manage a template",
		Options: []*discordgo.ApplicationCommandOption{{
			Name:         "template",
			Description:  "The template ID to manage",
			Type:         discordgo.ApplicationCommandOptionString,
			Required:     true,
			Autocomplete: true,
		}},
	}
}

func (c *TemplatesCommand) OnAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func (c *TemplatesCommand) OnCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	var d templateData
	if err := marshaller.Unmarshal(i.Interaction.ApplicationCommandData().Options, &d); err != nil {
		c.logger.Error("load-credentials-data", "error", err)
		ErrorResponse(s, i.Interaction, "Could not load data from interaction. Error: "+err.Error())
		return
	}

	template, err := c.templates.Find(d.Id)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching template details. Error: "+err.Error())
		return
	}
	embeds, components := templateEmbed(template)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		c.logger.Error("send-response", "error", err)
		return
	}
}

func valOrNotSet(v string) string {
	if v == "" {
		return "not set"
	}
	return v
}

func templateEmbed(s *resources.Template) (embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	embeds = append(embeds, &discordgo.MessageEmbed{
		Color: ColorDarkGrey,
		Title: s.Name,
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "ID",
			Value: s.TemplateId,
		}, {
			Name:   "Server Name Template",
			Value:  "`" + valOrNotSet(s.ServerNameTemplate) + "`",
			Inline: true,
		}, {
			Name:   "Welcome Message",
			Value:  "`" + valOrNotSet(s.WelcomeMessage) + "`",
			Inline: false,
		}, {
			Name:   "Autobalance Threshold",
			Value:  strconv.Itoa(s.AutoBalanceThreshold),
			Inline: true,
		}, {
			Name:   "Teamswitch cooldown",
			Value:  strconv.Itoa(s.TeamSwitchCooldown),
			Inline: true,
		}, {
			Name:   "Profanity filter",
			Value:  strings.Join(s.ProfanityFilter, "\n"),
			Inline: false,
		}},
	})
	components = append(components, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Update message and Server name Template",
				CustomID: customId(templatesPrefix, "set-messages", s.TemplateId),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label:    "Set Thresholds",
				CustomID: customId(templatesPrefix, "set-thresholds", s.TemplateId),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label:    "Set Profanity filter",
				CustomID: customId(templatesPrefix, "set-profanity-filter", s.TemplateId),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Emoji:    &discordgo.ComponentEmoji{ID: "1283790096461594655"},
				CustomID: customId(templatesPrefix, "refresh", s.TemplateId),
				Style:    discordgo.SecondaryButton,
			},
		}},
	}...)
	return
}

func (c *TemplatesCommand) onRefreshClick(s *discordgo.Session, i *discordgo.InteractionCreate, tplId string) {
	tpl, err := c.templates.Find(tplId)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching template details. Error: "+err.Error())
		return
	}
	embeds, components := templateEmbed(tpl)
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

func messagesModal(tpl resources.Template) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Title: "Set Messages",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "welcome-message",
					Label:    "Welcome Message",
					Style:    discordgo.TextInputParagraph,
					Value:    tpl.WelcomeMessage,
				},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "server-name-template",
					Label:    "Server Name Template",
					Style:    discordgo.TextInputShort,
					Value:    tpl.ServerNameTemplate,
				},
			}},
		},
		CustomID: customId(templatesPrefix, "confirm-messages", tpl.Id()),
	}
}

func thresholdsModal(tpl resources.Template) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Title: "Set Messages",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:  "team-switch-cooldown",
					Label:     "Teamswitch Cooldown (seconds)",
					Style:     discordgo.TextInputShort,
					Value:     strconv.Itoa(tpl.TeamSwitchCooldown),
					MaxLength: 3,
				},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:  "auto-balance-threshold",
					Label:     "Autobalance Threshold",
					Style:     discordgo.TextInputShort,
					Value:     strconv.Itoa(tpl.AutoBalanceThreshold),
					MaxLength: 2,
				},
			}},
		},
		CustomID: customId(templatesPrefix, "confirm-thresholds", tpl.Id()),
	}
}

func profanityFilterModal(tpl resources.Template) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Title: "Set Messages",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "profanity-filter",
					Label:    "Profanity (one per line)",
					Style:    discordgo.TextInputParagraph,
					Value:    strings.Join(tpl.ProfanityFilter, "\n"),
				},
			}},
		},
		CustomID: customId(templatesPrefix, "confirm-profanity-filter", tpl.Id()),
	}
}

type ModalDefinition func(tpl resources.Template) *discordgo.InteractionResponseData

func (c *TemplatesCommand) onSetModal(s *discordgo.Session, i *discordgo.InteractionCreate, tplId string, md ModalDefinition) {
	tpl, err := c.templates.Find(tplId)
	if err != nil {
		c.logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching template details. Error: "+err.Error())
		return
	}
	if tpl == nil {
		ErrorResponse(s, i.Interaction, "Could not find template with ID "+tplId)
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: md(*tpl),
	})
	if err != nil {
		c.logger.Error("message-component-respond", "error", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
	}
}

func (c *TemplatesCommand) OnMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.MessageComponentData().CustomID
	peek, _ := peekId(id)
	if matchesId(id, customId(templatesPrefix, "refresh")) {
		c.onRefreshClick(s, i, peek)
	} else if matchesId(id, customId(templatesPrefix, "set-messages")) {
		c.onSetModal(s, i, peek, messagesModal)
	} else if matchesId(id, customId(templatesPrefix, "set-thresholds")) {
		c.onSetModal(s, i, peek, thresholdsModal)
	} else if matchesId(id, customId(templatesPrefix, "set-profanity-filter")) {
		c.onSetModal(s, i, peek, profanityFilterModal)
	}
}

func (c *TemplatesCommand) OnModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.ModalSubmitData().CustomID
	peek, _ := peekId(id)
	if matchesId(id, customId(templatesPrefix, "confirm-messages")) {
		onConfirm(c.logger, c.templates, s, i, peek, func(tpl *resources.Template, d messagesData) {
			tpl.WelcomeMessage = d.WelcomeMessage
			tpl.ServerNameTemplate = d.ServerNameTemplate
		})
	} else if matchesId(id, customId(templatesPrefix, "confirm-thresholds")) {
		onConfirm(c.logger, c.templates, s, i, peek, func(tpl *resources.Template, d thresholdsData) {
			tpl.TeamSwitchCooldown = d.teamSwitchCooldown()
			tpl.AutoBalanceThreshold = d.autoBalanceThreshold()
		})
	} else if matchesId(id, customId(templatesPrefix, "confirm-profanity-filter")) {
		onConfirm(c.logger, c.templates, s, i, peek, func(tpl *resources.Template, d profanityData) {
			tpl.ProfanityFilter = d.ProfanityFilter()
		})
	}
}

type TemplateUpdate[T any] func(tpl *resources.Template, d T)

func onConfirm[T any](logger *slog.Logger, templates internal.Storage[resources.Template], s *discordgo.Session, i *discordgo.InteractionCreate, tplId string, update TemplateUpdate[T]) {
	tpl, err := templates.Find(tplId)
	if err != nil {
		logger.Error("find-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error fetching template details. Error: "+err.Error())
		return
	}
	if tpl == nil {
		ErrorResponse(s, i.Interaction, "Could not find template with ID "+tplId)
		return
	}
	var d T
	if err := marshaller.Unmarshal(i.ModalSubmitData().Components, &d); err != nil {
		logger.Error("parse-data", err)
		ErrorResponse(s, i.Interaction, "Unknown error: "+err.Error())
		return
	}
	update(tpl, d)
	err = templates.Save(*tpl)
	if err != nil {
		logger.Error("save-template", "error", err)
		ErrorResponse(s, i.Interaction, "There was an error saving the template. Error: "+err.Error())
		return
	}
	embeds, components := templateEmbed(tpl)
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
		},
	})
	if err != nil {
		logger.Error("edit-original-message", "error", err)
	}
}

func (c *TemplatesCommand) CanHandle(customId string) bool {
	return strings.HasPrefix(customId, templatesPrefix)
}
