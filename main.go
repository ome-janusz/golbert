package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/tkanos/gonfig"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Configuration struct {
	Message               string
	MembershipThreshold   int64
	NotificationChannelId string
	NotificationRoleId    string
}

type AuthConfiguration struct {
	Token string
}

var (
	configuration Configuration
	auth          AuthConfiguration
)

func init() {
	configuration = Configuration{}
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		fmt.Println("error reading config.json,", err)
		panic(err)
	}

	auth = AuthConfiguration{}
	err = gonfig.GetConf("auth.json", &auth)
	if err != nil {
		fmt.Println("error reading auth.json,", err)
		panic(err)
	}
}

func main() {
	fmt.Println("token: " + auth.Token)
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + auth.Token)
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

func getRole(session *discordgo.Session, guildId string, roleId string) (*discordgo.Role, error) {
	roles, err := session.GuildRoles(guildId)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == roleId {
			return role, nil
		}
	}
	return nil, fmt.Errorf("Role %s not found: ", roleId)
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
	s.ChannelMessageSend(m.ChannelID, "User joined at "+
		strconv.FormatInt(time.Now().Unix()-userJoinTime.Unix(), 10))
	msg := configuration.Message
	if len(configuration.NotificationRoleId) != 0 {
		role, err := getRole(s, m.GuildID, configuration.NotificationRoleId)
		if err != nil {
			panic(err)
		}
		msg = fmt.Sprintf("%s: %s", role.Mention(), msg)
	}
	s.ChannelMessageSend(m.ChannelID, msg)

	//if time.Now().Unix() - userJoinTime.Unix() < 10000 {
	s.ChannelMessageDelete(m.ChannelID, m.ID)
	//	s.GuildMemberDelete(m.GuildID, m.Author.ID)
	//}
}
