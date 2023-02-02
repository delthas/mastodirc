package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/delthas/mastodirc"
	"github.com/k3a/html2text"
	"github.com/mattn/go-mastodon"
	"gopkg.in/irc.v3"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

var cfg *mastodirc.Config

var lines = make(chan string, 1024)

func runMastodon() error {
	c := mastodon.NewClient(&mastodon.Config{
		Server:       cfg.MastodonServer,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AccessToken:  cfg.AccessToken,
	})
	if _, err := c.VerifyAppCredentials(context.Background()); err != nil {
		return fmt.Errorf("verifying credentials: %v", err)
	}

	go func() {
		var lastID mastodon.ID
		for {
			pg := &mastodon.Pagination{
				MinID: lastID,
			}
			if lastID == "" {
				pg.Limit = 1
			}
			statuses, err := c.GetTimelineHome(context.Background(), pg)
			if err != nil {
				log.Fatalf("getting timeling: %v", err)
			}
			if lastID != "" {
				if len(statuses) > 0 && statuses[len(statuses)-1].ID == lastID {
					statuses = statuses[:len(statuses)-1]
				}
				for i := len(statuses) - 1; i >= 0; i-- {
					status := statuses[i]
					var text string
					if status.Reblog != nil {
						text = fmt.Sprintf("RT @%v: %v", status.Reblog.Account.Acct, parseContent(status.Reblog.Content))
						for _, media := range status.Reblog.MediaAttachments {
							text += fmt.Sprintf("\n%v", media.URL)
						}
					} else {
						text = parseContent(status.Content)
						for _, media := range status.MediaAttachments {
							text += fmt.Sprintf("\n%v", media.URL)
						}
					}
					if text == "" {
						continue
					}
					prefix := fmt.Sprintf("<%v> ", status.Account.Username)
					first := true
					for _, line := range strings.Split(text, "\n") {
						if line == "" {
							continue
						}
						l := prefix + line
						if first {
							first = false
							prefix = strings.Repeat(" ", 3)
						}
						select {
						case lines <- l:
						default:
						}
					}
				}
			} else if len(statuses) > 0 {
				lastID = statuses[0].ID
			}
			if len(statuses) == 0 {
				time.Sleep(1 * time.Minute)
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}()
	return nil
}

func runIRC() error {
	var doTLS bool
	var host string
	if u, err := url.Parse(cfg.IRCServer); err == nil && u.Scheme != "" && u.Host != "" {
		switch u.Scheme {
		case "ircs":
			doTLS = true
		case "irc+insecure", "irc":
		default:
			return fmt.Errorf("invalid IRC addr scheme: %v", cfg.IRCServer)
		}
		host = u.Host
	} else if strings.Contains(cfg.IRCServer, ":+") {
		doTLS = true
		host = strings.ReplaceAll(cfg.IRCServer, ":+", ":")
	} else {
		host = cfg.IRCServer
	}
	go func() {
		var closeCh chan struct{}
		first := true
		for {
			if closeCh != nil {
				close(closeCh)
				closeCh = nil
			}
			if first {
				first = false
			} else {
				time.Sleep(10 * time.Second)
			}
			var nc net.Conn
			var err error
			if doTLS {
				nc, err = tls.Dial("tcp", host, nil)
			} else {
				nc, err = net.Dial("tcp", host)
			}
			if err != nil {
				log.Printf("connecting to irc: %v", err)
				continue
			}
			c := irc.NewClient(nc, irc.ClientConfig{
				Nick:      cfg.Nick,
				User:      cfg.Nick,
				Name:      cfg.Nick,
				SendLimit: 500 * time.Millisecond,
				SendBurst: 4,
				Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
					switch m.Command {
					case "001":
						c.WriteMessage(&irc.Message{
							Command: "JOIN",
							Params:  []string{cfg.Channel},
						})
					case "JOIN":
						if m.Name == c.CurrentNick() {
							closeCh = make(chan struct{}, 1)
							go func() {
								for {
									select {
									case line := <-lines:
										c.WriteMessage(&irc.Message{
											Command: "PRIVMSG",
											Params:  []string{cfg.Channel, line},
										})
									case <-closeCh:
										return
									}
								}
							}()
						}
					}
				}),
			})
			if err := c.Run(); err != nil {
				log.Printf("running irc: %v", err)
			}
		}
	}()
	return nil
}

func main() {
	path := flag.String("config", "mastodirc.yaml", "mastodirc configuration file path")
	flag.Parse()

	var err error
	cfg, err = mastodirc.ReadConfig(*path)
	if err != nil {
		log.Fatal(err)
	}

	if err := runMastodon(); err != nil {
		log.Fatal(err)
	}
	if err := runIRC(); err != nil {
		log.Fatal(err)
	}

	select {}
}

func parseContent(content string) string {
	text := html2text.HTML2TextWithOptions(content, html2text.WithUnixLineBreaks())
	text = strings.TrimSpace(text)
	return text
}
