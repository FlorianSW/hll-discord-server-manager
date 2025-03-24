package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/util"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"strconv"
)

func serversEmbed(s internal.Storage[resources.Server]) (embeds []*discordgo.MessageEmbed, buttons []discordgo.MessageComponent, err error) {
	buttons = append(buttons, discordgo.Button{
		Emoji:    &discordgo.ComponentEmoji{ID: "1283790096461594655"},
		Style:    discordgo.SecondaryButton,
		Disabled: false,
		CustomID: customId(createEmbedPrefix, "refresh"),
	})

	sl, err := s.List()
	if err != nil {
		return nil, nil, err
	}
	var servers []discordgo.SelectMenuOption
	for _, id := range sl {
		sd, err := s.Find(id)
		if err != nil {
			return nil, nil, err
		}
		servers = append(servers, discordgo.SelectMenuOption{
			Label: sd.Name,
			Value: sd.ServerId,
		})
	}

	return embeds, []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType: discordgo.StringSelectMenu,
					CustomID: customId(embedPrefix, "select-server"),
					Options:  servers,
				},
			},
		},
		discordgo.ActionsRow{
			Components: buttons,
		},
	}, nil
}

func serverEmbed(t internal.Storage[resources.Template], s resources.Server) (embeds []*discordgo.MessageEmbed, buttons []discordgo.MessageComponent, err error) {
	tc := tcadminClient(*s.TCAdminCredentials)
	si, err := tc.ServerInfo(s.TCAdminCredentials.ServiceId)

	if err != nil {
		return nil, nil, err
	}

	cc := crconClient(*s.CRConCredentials)
	pids, err := cc.PlayerIds(context.Background())
	if err != nil {
		return nil, nil, err
	}

	sl, err := t.List()
	if err != nil {
		return nil, nil, err
	}
	var templates []discordgo.SelectMenuOption
	for _, id := range sl {
		sd, err := t.Find(id)
		if err != nil {
			return nil, nil, err
		}
		templates = append(templates, discordgo.SelectMenuOption{
			Label: sd.Name,
			Value: sd.TemplateId,
		})
	}

	pu := resources.ServerUpdate{}
	if s.PendingUpdate != nil {
		pu = *s.PendingUpdate
	}
	templateName := "not set"
	for _, template := range templates {
		if template.Value == pu.TemplateId {
			templateName = template.Label
		}
	}
	serverName := si.Name
	if pu.ServerName != "" {
		serverName = fmt.Sprintf("~~%s~~ -> %s", serverName, pu.ServerName)
	}
	serverPassword := si.Password
	if pu.ServerPassword != "" {
		serverPassword = fmt.Sprintf("~~%s~~ -> %s", serverPassword, pu.ServerPassword)
	}
	embeds = append(embeds, &discordgo.MessageEmbed{
		Title:       s.Name,
		Description: "See the server details below. You can change details, which are only applied when you confirm the changes. The server might then be restarted!",
		Color:       util.ColorDarkBlue,
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "Player Count",
			Value: strconv.Itoa(len(pids)),
		}, {
			Name:  "Template",
			Value: templateName,
		}, {
			Name:   "Server Name",
			Value:  serverName,
			Inline: true,
		}, {
			Name:   "Server Password",
			Value:  serverPassword,
			Inline: true,
		}},
	})
	buttons = append(buttons, []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Save and restart",
			Style:    discordgo.PrimaryButton,
			Disabled: false,
			CustomID: customId(embedPrefix, "save-restart", s.ServerId),
		}, discordgo.Button{
			Label:    "Set Name & Password",
			Style:    discordgo.SecondaryButton,
			Disabled: false,
			CustomID: customId(embedPrefix, "set-name-password", s.ServerId),
		}, discordgo.Button{
			Emoji:    &discordgo.ComponentEmoji{ID: "1283790096461594655"},
			Style:    discordgo.SecondaryButton,
			Disabled: false,
			CustomID: customId(embedPrefix, "refresh", s.ServerId),
		}}...)

	return embeds, []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType: discordgo.StringSelectMenu,
					CustomID: customId(embedPrefix, "select-template", s.ServerId),
					Options:  templates,
				},
			},
		},
		discordgo.ActionsRow{
			Components: buttons,
		},
	}, nil
}
