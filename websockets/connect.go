package websockets

import (
	"crypto/sha1"
	"encoding/base64"
	"net"
	"os"
)

func GenerateKey(clientKey string) string {
	secretKey := os.Getenv("WEBSOCKETS_SECRET_KEY")
	hash := sha1.Sum(append([]byte(clientKey), secretKey...))
	return base64.StdEncoding.EncodeToString(hash[:])
}

var connections []Connection

type Connection struct {
	ClientKey string
	Conn      net.Conn
	running   chan bool
	input     chan string
}

func NewConnection(hijackedConn net.Conn, clientKey string) *Connection {
	conn := Connection{
		Conn:      hijackedConn,
		ClientKey: clientKey,
		running:   make(chan bool),
		input:     make(chan string),
	}
	connections = append(connections, conn)
	return &conn
}

func (conn Connection) Close() {
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
		}
	}
	conn.Conn.Close()
	close(conn.running)
}

func (conn Connection) HandleConnection() {
	go conn.transferMessages()
	<-conn.running
}
