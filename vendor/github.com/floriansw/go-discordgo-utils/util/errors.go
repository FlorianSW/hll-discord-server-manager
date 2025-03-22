package util

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"net/http"
	"strconv"
)

func hasResponse(s *discordgo.Session, i *discordgo.Interaction) bool {
	_, err := s.InteractionResponse(i)
	var rerr *discordgo.RESTError
	if errors.As(err, &rerr) {
		if rerr.Response.StatusCode == http.StatusNotFound {
			return false
		}
		println("Unexpected status code while retrieving interaction response for error response: " + strconv.Itoa(rerr.Response.StatusCode))
		return false
	}
	if err != nil {
		println("Unexpected error checking for existing response to send error response: " + err.Error())
		return false
	}
	return true
}

func ErrorResponse(s *discordgo.Session, i *discordgo.Interaction, msg string) {
	var err error
	if hasResponse(s, i) {
		_, err = s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
			Content: &msg,
		})
	} else {
		err = s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
	if err != nil {
		println("Could not update response with message: " + err.Error())
	}
}
