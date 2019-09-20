module gitlab.com/Cacophony/Gateway

require (
	github.com/bwmarrin/discordgo v0.16.1-0.20190608205439-347a4f69b0b5
	github.com/getsentry/raven-go v0.2.0
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.8.1
	gitlab.com/Cacophony/go-kit v0.0.0-20190920182210-5e77ad481839
	go.uber.org/zap v1.10.0
)

replace gitlab.com/Cacophony/go-kit => ../go-kit

go 1.13
