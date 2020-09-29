package main

import (
    "regexp"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/tkanos/gonfig"
	"os"
	"os/signal"
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
	re			  *regexp.Regexp
)

func init() {
	configuration = Configuration{}
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		fmt.Println("error reading config.json,", err)
		panic(err)
	}
	
	// making the configuration compatible with the Node.js version
	configuration.MembershipThreshold *= 1000

	auth = AuthConfiguration{}
	err = gonfig.GetConf("auth.json", &auth)
	if err != nil {
		fmt.Println("error reading auth.json,", err)
		panic(err)
	}
	
	re = regexp.MustCompile("([a-zA-Z0-9]+://)?([a-zA-Z0-9_]+:[a-zA-Z0-9_]+@)?([a-zA-Z0-9.-]+\\.[A-Za-z]{2,4})(:[0-9]+)?(/.*)?")
}

func main() {
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

func Role(session *discordgo.Session, guildId string, roleId string) (*discordgo.Role, error) {
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

func UserJoinTime(s *discordgo.Session, guildId string, authorId string) (int64, error) {
	member, err := s.GuildMember(guildId, authorId)
	if err != nil {
		return 0, fmt.Errorf("error retrieving membership information,", err)
	}
	userJoinTime, err := member.JoinedAt.Parse()
	if err != nil {
		return 0, fmt.Errorf("error parsing user join time info,", err)
	}
	return userJoinTime.UnixNano(), nil
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself or empty state updates
	if m.Author.ID == s.State.User.ID || len(m.Content) == 0 {
		return
	}

	user, err := s.User(m.Author.ID)
    if err != nil {
        fmt.Println("error retrieving user information of user %s,",
            m.Author.ID, err)
		return
    }
    
	// also return if the message is posted by a bot
	if user.Bot {
        return
    }

    userJoinTime, err := UserJoinTime(s, m.GuildID, m.Author.ID)
	if err != nil {
		fmt.Println(err)
		return
	}

	if time.Now().UnixNano() - userJoinTime <  configuration.MembershipThreshold &&
			re.FindStringIndex(m.Content) != nil {
        s.ChannelMessageDelete(m.ChannelID, m.ID)
        s.GuildMemberDelete(m.GuildID, m.Author.ID)

        if len(configuration.NotificationChannelId) != 0 {
            s.ChannelMessageSend(configuration.NotificationChannelId,
                fmt.Sprintf("Linkspam geplaatst door gebruiker <@%s>; gebruiker wordt gekickt.", m.Author.ID))
        }
        
        msg := configuration.Message
        if len(configuration.NotificationRoleId) != 0 {
            role, err := Role(s, m.GuildID, configuration.NotificationRoleId)
            if err != nil {
                fmt.Println(err)
            }
            msg = fmt.Sprintf("%s: %s", role.Mention(), msg)
        }
        s.ChannelMessageSend(m.ChannelID, msg)
	}
}
