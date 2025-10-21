package client

func (cli *Client) Shutdown() error {
	if cli.Node.Host != nil {
		_ = cli.Node.Host.Close() // close the libp2p host
	}
	if cli.Node.DHT != nil {
		cli.Node.DHT.Close() // close the DHT
	}

	/* TODO: graceful pubsub shutdown

	// Close individual topics if they exist
	if cli.Node.Topics.ChatTopic != nil {
		_ = cli.Node.Topics.ChatTopic.Close()
	}
	if cli.Node.Topics.RekeyTopic != nil {
		_ = cli.Node.Topics.RekeyTopic.Close()
	}
	// Add any other topics that need to be closed
	*/
	if cli.Node.Ctx != nil {
		cli.Node.Ctx.Done() // cancel the context
	}

	return nil

}
