package websockets

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"log"
	"net"
	"os"
	"strings"
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
	go conn.receiveMessage()
	go conn.sendMessage()
	<-conn.running
}

func (conn Connection) receiveMessage() {
	for {
		select {
		case <-conn.running:
			return
		default:
			frame, err := ReadFrame(conn.Conn)
			if err != nil {
				switch err.(type) {
				case *UnmaskedFrame:
					closeFrame := Frame{
						FIN:     true,
						Opcode:  0x8,
						Payload: []byte{0b11, 0b11101010}, // 1002
					}
					if _, err := conn.Conn.Write(closeFrame.Bytes()); err != nil {
						log.Println(err.Error())
					}
					conn.Close()
				default:
					log.Println(err.Error())
					conn.Close()
				}
			}

			message, err := frame.Decode()
			if err != nil {
				log.Println(err.Error())
				conn.Close()
			}

			frameToClient := Frame{
				FIN:     true,
				Opcode:  0x1,
				Payload: []byte(message),
			}
			for _, c := range connections {
				if c != conn {
					if _, err := c.Conn.Write(frameToClient.Bytes()); err != nil {
						log.Println(err.Error())
						conn.Close()
					}
				}
			}
		}
	}
}

func (conn Connection) sendMessage() {
	go conn.readUserInput()
	for {
		select {
		case <-conn.running:
			return
		default:
			message := <-conn.input
			frame := Frame{
				FIN:     true,
				Opcode:  0x1,
				Payload: []byte(message),
			}

			if _, err := conn.Conn.Write(frame.Bytes()); err != nil {
				log.Println(err.Error())
				conn.Close()
			}
		}
	}
}

func (conn Connection) readUserInput() {
	for {
		message, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println(err.Error())
			conn.Close()
		}
		conn.input <- strings.TrimSpace(message)
	}
}
