package mikrotik

import (
	"fmt"

	routeros "github.com/go-routeros/routeros/v3"
)

type Client struct {
	Address  string
	Username string
	Password string
	conn     *routeros.Client
}

func NewClient(address, username, password string) *Client {
	return &Client{
		Address:  address,
		Username: username,
		Password: password,
	}
}

func (c *Client) Connect() error {
	conn, err := routeros.Dial(c.Address, c.Username, c.Password)
	if err != nil {
		return fmt.Errorf("can't connect to Mikrotik(%s): %w", c.Address, err)
	}
	c.conn = conn
	return nil
}

func (c *Client) Disconnect() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) IsConnect() bool {
	return c.conn != nil
}
