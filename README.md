# mastodirc [![builds.sr.ht status](https://builds.sr.ht/~delthas/mastodirc.svg)](https://builds.sr.ht/~delthas/mastodirc)

A simple bot that copies your Mastodon home timeline to an IRC channel, live.

## Usage

Copy and edit `mastodirc.yaml.sample` into `mastodirc.yaml`

Create & authorize a Mastodon app:
```shell
go run ./cmd/mastodirc-init -server <mastodon_server_url>
```

This will automatically update `mastodirc.yaml` with the appropriate tokens.

Then, run:
```shell
go run ./cmd/mastodirc
```

Statuses from your home timeline will be sent to the IRC channel.

## License

MIT
