package handler

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"log/slog"
)

type discordApp struct {
	logger          *slog.Logger
	session         *discordgo.Session
	guildId         string
	commands        []*discordgo.ApplicationCommand
	commandHandlers map[string]Command
}

func New(logger *slog.Logger, session *discordgo.Session, guildId string, commands map[string]Command) *discordApp {
	handler := &discordApp{
		logger:   logger,
		session:  session,
		guildId:  guildId,
		commands: []*discordgo.ApplicationCommand{},
	}

	handler.commandHandlers = commands
	for cmd, command := range handler.commandHandlers {
		handler.commands = append(handler.commands, command.Definition(cmd))
	}

	return handler
}

func containsCommand(c []*discordgo.ApplicationCommand, cmd string) bool {
	for _, command := range c {
		if command.Name == cmd {
			return true
		}
	}
	return false
}

func (a *discordApp) Listen() error {
	cmds, err := a.session.ApplicationCommands(a.session.State.User.ID, a.guildId)
	if err != nil {
		return err
	}
	for _, command := range cmds {
		if !containsCommand(a.commands, command.Name) {
			if err := a.session.ApplicationCommandDelete(a.session.State.User.ID, a.guildId, command.ID); err != nil {
				a.logger.Error("delete-command", err, "name", command.Name)
			}
		}
	}

	for _, v := range a.commands {
		if containsCommand(cmds, v.Name) {
			continue
		}
		_, err := a.session.ApplicationCommandCreate(a.session.State.User.ID, a.guildId, v)
		if err != nil {
			a.logger.Error("create-command", err, "command", v)
		}
	}

	a.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID != a.guildId {
			a.error(s, i.Interaction, "The command is not available for your discord server.")
			return
		}
		var (
			name string
			h    Command
			mc   MessageComponent
			ok   bool
		)
		switch i.Type {
		case discordgo.InteractionApplicationCommandAutocomplete:
			fallthrough
		case discordgo.InteractionApplicationCommand:
			name = i.ApplicationCommandData().Name
			if h, ok = a.commandHandlers[name]; !ok {
				a.error(s, i.Interaction, "Command does not exist: "+name)
				return
			}
		case discordgo.InteractionMessageComponent:
			cid := i.MessageComponentData().CustomID
			for cmd, command := range a.commandHandlers {
				if cast, ok := command.(MessageComponent); ok && cast.CanHandle(cid) {
					name = cmd
					mc = cast
				}
			}
			if mc == nil {
				a.error(s, i.Interaction, "Command does not support message component: "+cid)
				return
			}
		case discordgo.InteractionModalSubmit:
			cid := i.ModalSubmitData().CustomID
			a.logger.Info("modalsubmit", "custom_id", cid)
			for _, command := range a.commandHandlers {
				if cast, ok := command.(ModalSubmit); ok && cast.CanHandle(cid) {
					cast.OnModalSubmit(s, i)
					return
				}
			}
			a.error(s, i.Interaction, "Command does not support modal submit: "+cid)
			return
		default:
			a.logger.Error("unhandled-interaction", errors.New("unhandled: "+i.Type.String()))
			a.error(s, i.Interaction, "unhandled interaction type: "+i.Type.String())
			return
		}

		switch i.Type {
		case discordgo.InteractionApplicationCommandAutocomplete:
			a.logger.Info("autocomplete", "name", name)
			if ac, tok := h.(Autocomplete); tok {
				ac.OnAutocomplete(s, i)
			} else {
				a.error(s, i.Interaction, "Command does not support autocomplete: "+name)
			}
		case discordgo.InteractionMessageComponent:
			a.logger.Info("messagecomponent", "name", name)
			mc.OnMessageComponent(s, i)
		case discordgo.InteractionApplicationCommand:
			a.logger.Info("command", "name", name)
			h.OnCommand(s, i)
		}
	})
	return nil
}

func (a *discordApp) error(s *discordgo.Session, i *discordgo.Interaction, msg string) {
	s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
