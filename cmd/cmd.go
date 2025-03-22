package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/floriansw/go-discordgo-utils/handler"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/internal/commands"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	level := slog.LevelInfo
	if _, ok := os.LookupEnv("DEBUG"); ok {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	c, err := internal.NewConfig("./config.json", logger)
	if err != nil {
		logger.Error("config", err)
		return
	}

	var s *discordgo.Session
	if c.Discord != nil {
		s, err = discordgo.New("Bot " + c.Discord.Token)
		if err != nil {
			logger.Error("discord", err)
			return
		}
	}
	if err = os.MkdirAll("./servers/", 0644); err != nil {
		logger.Error("create-matches", err)
		return
	}
	servers := resources.NewServers("./servers/")
	h := handler.New(logger, s, c.Discord.GuildID, map[string]handler.Command{
		"create-embed": commands.NewCreateEmbedCommand(logger, c, servers),
		"add-server":   commands.NewAddServerCommand(logger, c, servers),
		"credentials":  commands.NewCredentialsCommand(logger, c, servers),
	})
	if s != nil {
		s.AddHandlerOnce(func(s *discordgo.Session, e *discordgo.Ready) {
			if err := h.Listen(); err != nil {
				logger.Error("discord-listen", err)
				panic(err)
			}
			logger.Info("ready")
		})
		err = s.Open()
		if err != nil {
			logger.Error("open-session", err)
			return
		}
		defer s.Close()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("graceful-shutdown")
	if err := c.Save(); err != nil {
		logger.Error("save-config", "error", err)
	}
}
