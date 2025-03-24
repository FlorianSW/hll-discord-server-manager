package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
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
