package main

import "hillside/internal/models"


func (cli *Client) requetServers() (*models.ListServersResponse, error) {
	var listResp models.ListServersResponse
	err := cli.Node.SendRPC("ListServers", models.ListServersRequest{}, &listResp)
	if err != nil {
		return nil, err
	}
	return &listResp, nil

}