package websockets

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"os"
	"reflect"
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
	queue     FrameQueue
}

func NewConnection(hijackedConn net.Conn, clientKey string) *Connection {
	conn := Connection{
		Conn:      hijackedConn,
		ClientKey: clientKey,
		queue:     FrameQueue{},
	}
	connections = append(connections, conn)
	return &conn
}

func (conn Connection) Close() {
	for i, c := range connections {
		if reflect.DeepEqual(c, conn) {
			connections = append(connections[:i], connections[i+1:]...)
		}
	}
	conn.Conn.Close()
}

func (conn Connection) HandleConnection() {
Main:
	for {
		frame, err := ReadFrame(conn.Conn)
		if err != nil {
			log.Println(err.Error())
			conn.Close()
			break Main
		}

		if frame.Opcode == 0x8 {
			message, err := frame.Decode()
			if err != nil {
				log.Println(err.Error())
				conn.Close()
				break Main
			}
			statusCode := int(message[0])<<8&0xFF00 + int(message[1])
			conn.sendCloseFrame(statusCode)
			conn.Close()
			continue
		}

		if !frame.IsMasked {
			log.Println(errors.New("frame isn't masked").Error())
			conn.Close()
			break Main
		}

		conn.queue.Enqueue(frame)

		var opcode byte
		var message string
		if frame.FIN {
			opcode = conn.queue[0].Opcode
			messageParts := []string{}
			for {
				continuationFrame, ok := conn.queue.Dequeue()
				if !ok {
					break
				}
				continuationFrame.FIN = true
				continuationFrame.Opcode = opcode
				messagePart, err := continuationFrame.Decode()
				if err != nil {
					log.Println(err.Error())
					conn.Close()
					break Main
				}
				messageParts = append(messageParts, messagePart)
			}
			message = strings.Join(messageParts, "")
		} else {
			continue
		}

		frameToClient := Frame{
			FIN:     true,
			Opcode:  opcode,
			Payload: []byte(message),
		}
		for _, c := range connections {
			if !reflect.DeepEqual(c, conn) {
				c.Conn.Write(frameToClient.Bytes())
			}
		}
	}
}
