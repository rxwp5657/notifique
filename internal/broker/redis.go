package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/notifique/internal/server/dto"

	redis "github.com/redis/go-redis/v9"
)

type BrokerConfigurator interface {
	GetBrokerChannelSize() (int, error)
}

type userChannel struct {
	NotificationCh chan dto.UserNotification
	Quit           chan bool
}

type BrokerRedisApi interface {
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
}

type Redis struct {
	client          BrokerRedisApi
	channels        map[string]userChannel
	channelCapacity int
}

func (rb *Redis) Suscribe(ctx context.Context, userId string) (<-chan dto.UserNotification, error) {

	if rb == nil {
		return nil, fmt.Errorf("redis broker is nil")
	}

	if ch, ok := rb.channels[userId]; ok {
		return ch.NotificationCh, nil
	}

	userCh := userChannel{
		NotificationCh: make(chan dto.UserNotification, rb.channelCapacity),
		Quit:           make(chan bool),
	}

	rb.channels[userId] = userCh

	pubsub := rb.client.Subscribe(ctx, userId)

	go func() {
		defer pubsub.Close()
		redisCh := pubsub.Channel()

		for {
			select {
			case <-userCh.Quit:
				return
			case msg := <-redisCh:
				un := dto.UserNotification{}
				err := json.Unmarshal([]byte(msg.Payload), &un)
				if err != nil {
					slog.Error(err.Error())
				}
				userCh.NotificationCh <- un
			}
		}
	}()

	return userCh.NotificationCh, nil
}

func (rb *Redis) Unsubscribe(ctx context.Context, userId string) error {

	if rb == nil {
		return fmt.Errorf("redis broker is nil")
	}

	if ch, ok := rb.channels[userId]; ok {
		ch.Quit <- true
		close(ch.NotificationCh)
		close(ch.Quit)

		delete(rb.channels, userId)
	}

	return nil
}

func (rb *Redis) Publish(ctx context.Context, userId string, un dto.UserNotification) error {

	if rb == nil {
		return fmt.Errorf("redis broker is nil")
	}

	marshalled, err := json.Marshal(un)

	if err != nil {
		return fmt.Errorf("failed to marshall user notificaion - %w", err)
	}

	if err = rb.client.Publish(ctx, userId, string(marshalled)).Err(); err != nil {
		return fmt.Errorf("failed to publish user notification - %w", err)
	}

	return nil
}

func NewRedisBroker(client BrokerRedisApi, bc BrokerConfigurator) (*Redis, error) {

	channelSize, err := bc.GetBrokerChannelSize()

	if err != nil {
		return nil, err
	}

	if channelSize == 0 {
		return nil, fmt.Errorf("broker channel size must be > 0")
	}

	broker := &Redis{
		client:          client,
		channels:        make(map[string]userChannel),
		channelCapacity: channelSize,
	}

	return broker, nil
}
