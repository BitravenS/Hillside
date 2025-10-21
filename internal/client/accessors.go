package client

import "fmt"

func (cli *Client) GetServerName() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Server == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Server.ServerMeta.Name
}

func (cli *Client) GetServerID() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Server == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Server.ServerMeta.ID
}

func (cli *Client) GetRoomName() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Room == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Room.RoomMeta.Name
}

func (cli *Client) GetRoomID() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Room == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Room.RoomMeta.ID
}
