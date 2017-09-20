package nats

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"time"
)

func NewContact(identity dto_discovery.Identity) dto_discovery.Contact {
	return dto_discovery.Contact{
		Type: CONTACT_NATS_V1,
		Definition: ContactNATSV1{
			Topic: string(identity),
		},
	}
}

func NewServer() *channelNats {
	return &channelNats{
		options:        getDefaultOptions(),
		timeoutRequest: 500 * time.Millisecond,
	}
}

func NewClient() *channelNats {
	return &channelNats{
		options:        getDefaultOptions(),
		timeoutRequest: 500 * time.Millisecond,
	}
}

func getDefaultOptions() nats.Options {
	options := nats.GetDefaultOptions()
	options.Servers = []string{
		"nats://127.0.0.1:4222",
	}
	return options
}
