package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/delthas/mastodirc"
	"github.com/mattn/go-mastodon"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
)

const redirectURIOOB = "urn:ietf:wg:oauth:2.0:oob"

var cfg *mastodirc.Config

func main() {
	config := flag.String("config", "mastodirc.yaml", "mastodirc configuration file path")
	server := flag.String("server", "", "mastodon server URL")
	flag.Parse()

	var err error
	cfg, err = mastodirc.ReadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}

	var authURI string
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		if *server != "" {
			cfg.MastodonServer = *server
		}
		if cfg.MastodonServer == "" {
			flag.Usage()
			os.Exit(1)
		}
		app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
			Server:     cfg.MastodonServer,
			ClientName: "mastodirc",
			Scopes:     "read",
		})
		if err != nil {
			log.Fatal(err)
		}
		cfg.ClientID = app.ClientID
		cfg.ClientSecret = app.ClientSecret
		authURI = app.AuthURI
	} else {
		u, err := url.Parse(cfg.MastodonServer)
		if err != nil {
			log.Fatal(err)
		}
		u.Path = path.Join(u.Path, "/oauth/authorize")
		u.RawQuery = url.Values{
			"scope":         {"read"},
			"response_type": {"code"},
			"redirect_uri":  {redirectURIOOB},
			"client_id":     {cfg.ClientID},
		}.Encode()
		authURI = u.String()
	}
	fmt.Println("Open the URL below in your browser:")
	fmt.Println(authURI)
	fmt.Print("Enter the authorization code: ")
	var code string
	for code == "" {
		if _, err := fmt.Scanln(&code); err != nil {
			log.Fatal(err)
		}
		code = strings.TrimSpace(code)
	}
	c := mastodon.NewClient(&mastodon.Config{
		Server:       cfg.MastodonServer,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	})
	if err := c.AuthenticateToken(context.Background(), code, redirectURIOOB); err != nil {
		log.Fatal(err)
	}
	cfg.AccessToken = c.Config.AccessToken

	if err := mastodirc.WriteConfig(*config, cfg); err != nil {
		log.Fatal(err)
	}
}
