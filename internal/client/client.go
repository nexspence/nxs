package client

type Client struct{}

func New(url, token string) *Client { return &Client{} }
