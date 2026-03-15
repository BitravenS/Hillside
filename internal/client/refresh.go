package client

import (
	"context"
	"encoding/json"
	"time"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/utils"
)

func (cli *Client) refreshRoomList() {
	roomResp, err := cli.requestRooms(cli.GetServerID())
	cli.UI.App.QueueUpdateDraw(func() {
		if err != nil {
			cli.UI.ShowError("Server Error", err.Error(), "Go back to Browse view", 0, func() {
				cli.UI.Pages.SwitchToPage("browse")
			})
			return
		} else {
			cli.UI.ChatScreen.UpdateRoomList(roomResp.Rooms)
		}
	})
}
func (cli *Client) refreshServerList() {
	serverResp, err := cli.requestServers()
	cli.UI.App.QueueUpdateDraw(func() {
		if err != nil {
			cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
				cli.UI.Pages.SwitchToPage("login")
			})
			return
		} else {
			cli.UI.BrowseScreen.UpdateServerList(serverResp.Servers)
		}
	})
}

func (cli *Client) StartServerAutoRefresh() {
	cli.refreshServerList()
	currentPage, _ := cli.UI.Pages.GetFrontPage()
	cli.Session.Log.Logf("Starting auto-refresh on page: %s", currentPage)

	// cancellable context for this auto-refresh session
	refreshCtx, cancelRefresh := context.WithCancel(cli.Node.Ctx)
	defer cancelRefresh()

	go cli.RefreshTicker(500*time.Millisecond, refreshCtx, cancelRefresh, utils.IsBrowsePageActive)

	ServersTopic := p2p.ServersTopic()
	top, err := cli.Node.PS.Join(ServersTopic)
	if err != nil {
		cli.Session.Log.Logf("Failed to join servers topic: %v", err)
		return
	}

	sub, err := top.Subscribe()
	if err != nil {
		cli.Session.Log.Logf("Failed to subscribe to servers topic: %v", err)
		return
	}
	defer sub.Cancel()

	for {
		select {
		case <-refreshCtx.Done():
			cli.Session.Log.Logf("Auto-refresh cancelled")
			return
		default:
			msg, err := sub.Next(refreshCtx)
			if err != nil {
				if err == context.Canceled {
					cli.Session.Log.Logf("Auto-refresh stopped due to page change")

					currentPage, _ := cli.UI.Pages.GetFrontPage()
					cli.Session.Log.Logf("Current page: %s", currentPage)
				} else {
					cli.Session.Log.Logf("Error reading from servers topic: %v", err)
				}
				return
			}

			var listResp models.ListServersResponse
			err = json.Unmarshal(msg.Data, &listResp)
			if err != nil {
				cli.Session.Log.Logf("Error unmarshaling servers topic message: %v", err)

			}
			cli.Session.Log.Logf("Received server list update with %d servers", len(listResp.Servers))
			cli.UI.App.QueueUpdateDraw(func() {
				if err != nil {
					cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
						cli.UI.Pages.SwitchToPage("login")
					})
					return
				} else {
					cli.UI.BrowseScreen.UpdateServerList(listResp.Servers)
				}
			})
		}
	}
}
func (cli *Client) StartRoomAutoRefresh() {
	cli.refreshServerList()
	currentPage, _ := cli.UI.Pages.GetFrontPage()
	cli.Session.Log.Logf("Starting auto-refresh on page: %s", currentPage)

	refreshCtx, cancelRefresh := context.WithCancel(cli.Node.Ctx)
	defer cancelRefresh()

	go cli.RefreshTicker(500*time.Millisecond, refreshCtx, cancelRefresh, utils.IsChatPageActive)

	RoomsTopic := p2p.RoomsTopic(cli.GetServerID())
	err := setupServerTopics(cli, models.TopicRooms, RoomsTopic)
	if err != nil {
		cli.Session.Log.Logf("Failed to setup rooms topic: %v", err)
		return
	}
	cli.Session.Log.Logf("Subscribed to rooms topic for server %s", cli.GetServerID())

	sub, err := subToServerTopic(cli, models.TopicRooms)
	if err != nil {
		cli.Session.Log.Logf("Failed to subscribe to rooms topic: %v", err)
		return
	}
	defer sub.Cancel()

	for {
		select {
		case <-refreshCtx.Done():
			cli.Session.Log.Logf("Auto-refresh cancelled")
			return
		default:
			msg, err := sub.Next(refreshCtx)
			if err != nil {
				if err == context.Canceled {
					cli.Session.Log.Logf("Auto-refresh stopped due to page change")

					currentPage, _ := cli.UI.Pages.GetFrontPage()
					cli.Session.Log.Logf("Current page: %s", currentPage)
				} else {
					cli.Session.Log.Logf("Error reading from rooms topic: %v", err)
				}
				return
			}

			var listResp models.ListRoomsResponse
			err = json.Unmarshal(msg.Data, &listResp)
			if err != nil {
				cli.Session.Log.Logf("Error unmarshaling rooms topic message: %v", err)

			}
			cli.Session.Log.Logf("Received rooms list update with %d rooms", len(listResp.Rooms))
			cli.UI.App.QueueUpdateDraw(func() {
				if err != nil {
					cli.UI.ShowError("Rooms Error", err.Error(), "Go back to servers", 0, func() {
						cli.UI.Pages.SwitchToPage("browse")
					})
					return
				} else {
					cli.UI.ChatScreen.UpdateRoomList(listResp.Rooms)
				}
			})
		}
	}
}

func (cli *Client) RefreshTicker(d time.Duration, refreshCtx context.Context, cancelRefresh context.CancelFunc, isActiveFunc func(string) bool) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentPage, _ := cli.UI.Pages.GetFrontPage()
			if !isActiveFunc(currentPage) {
				cli.Session.Log.Logf("Left chat page, stopping auto-refresh")
				cancelRefresh()
				return
			}
		case <-refreshCtx.Done():
			return
		}
	}
}
