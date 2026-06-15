package mcp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type streamTransport struct {
	rwc io.ReadWriteCloser
}

type streamConn struct {
	rwc io.ReadWriteCloser
	r   *bufio.Reader
	mu  sync.Mutex
}

func NewStreamTransport(rwc io.ReadWriteCloser) mcpsdk.Transport {
	return &streamTransport{rwc: rwc}
}

func (t *streamTransport) Connect(context.Context) (mcpsdk.Connection, error) {
	return &streamConn{rwc: t.rwc, r: bufio.NewReader(t.rwc)}, nil
}

func (c *streamConn) Read(context.Context) (jsonrpc.Message, error) {
	data, err := c.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return jsonrpc.DecodeMessage(data[:len(data)-1])
}

func (c *streamConn) Write(_ context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err1 := c.rwc.Write(data)
	_, err2 := c.rwc.Write([]byte{'\n'})
	return errors.Join(err1, err2)
}

func (c *streamConn) Close() error {
	return c.rwc.Close()
}

func (c *streamConn) SessionID() string {
	return ""
}
