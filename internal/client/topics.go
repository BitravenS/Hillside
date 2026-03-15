package client

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"hillside/internal/models"
)

func setupRoomTopics(cli *Client, topic models.TopicName, topicNamespace string) error {
	if !cli.Session.Current.Room.Topics.HasTopic(topic) {
		top, err := cli.Node.PS.Join(topicNamespace)
		if err != nil {
			return err
		}
		cli.Session.Current.Room.Topics.SetTopic(topic, top)
	}
	return nil
}

func subToRoomTopic(cli *Client, topic models.TopicName) (sub *pubsub.Subscription, err error) {
	return cli.Session.Current.Room.Topics.GetTopic(topic).Subscribe()
}
func subToServerTopic(cli *Client, topic models.TopicName) (sub *pubsub.Subscription, err error) {
	return cli.Session.Current.Server.Topics.GetTopic(topic).Subscribe()
}

func setupServerTopics(cli *Client, topic models.TopicName, topicNamespace string) error {
	if !cli.Session.Current.Server.Topics.HasTopic(topic) {
		top, err := cli.Node.PS.Join(topicNamespace)
		if err != nil {
			return err
		}
		cli.Session.Current.Server.Topics.SetTopic(topic, top)
	}
	return nil
}

func (cli *Client) unsubscribeActiveSubs() {
	for _, sub := range cli.Node.Subs {
		if sub != nil {
			sub.Cancel()
		}
	}
	cli.Node.Subs = cli.Node.Subs[:0]
}
