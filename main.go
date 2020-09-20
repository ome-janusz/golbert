package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Variables used for command line parameters
var (
	Token string
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID ||
			len(m.Content) == 0 {
		return
	}

	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		fmt.Println("error retrieving membership information,", err)
		return
	}
	userJoinTime, _ := member.JoinedAt.Parse()
	s.ChannelMessageSend(m.ChannelID, "User joined at " +
		strconv.FormatInt(time.Now().Unix() - userJoinTime.Unix(), 10))

	if time.Now().Unix() - userJoinTime.Unix() < 10000 {
		s.GuildMemberDelete(m.GuildID, m.Author.ID)
	}
}
