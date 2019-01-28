package mocks

import "github.com/ethereum/go-ethereum/statediff/builder"

type Publisher struct {
	StateDiff      *builder.StateDiff
	publisherError error
}

func (publisher *Publisher) PublishStateDiff(sd *builder.StateDiff) (string, error) {
	publisher.StateDiff = sd
	return "", publisher.publisherError
}

func (publisher *Publisher) SetPublisherError(err error) {
	publisher.publisherError = err
}
