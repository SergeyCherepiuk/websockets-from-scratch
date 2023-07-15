package websockets

import (
	"crypto/sha1"
	"encoding/base64"
	"log"
	"net"
	"os"
	"sync"
)

var wg sync.WaitGroup

type Connection struct {
	Key string
}

func NewConnection(key string) Connection {
	return Connection{Key: key}
}

func (conn Connection) GenerateKey() string {
	secretKey := os.Getenv("WEBSOCKETS_SECRET_KEY")
	hash := sha1.Sum(append([]byte(conn.Key), secretKey...))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (conn Connection) HandleCommunication(c net.Conn) {
	defer c.Close()
	defer wg.Wait()

	wg.Add(1)
	go receiveMessage(c)
}

func receiveMessage(c net.Conn) {
	defer wg.Done()

	for {
		frame, err := ReadFrame(c)
		if err != nil {
			log.Println(err.Error())
			return
		}

		message, err := frame.Decode()
		if err != nil {
			log.Println(err.Error())
			return
		}

		log.Println(message)
	}
}

// func createFrame(message string, payloadLength []byte) []byte {
// 	frame := make([]byte, 0, 1+len(payloadLength)+len(message))
// 	frame = append(frame, 0b10000001)
// 	frame = append(frame, payloadLength...)
// 	return append(frame, message...)
// }

// // Common functions for both receiving and sending messages
// func createCloseFrame(message string, statusCode uint16) []byte {
// 	frame := make([]byte, 0, 2+2+len(message))
// 	frame = append(frame, 0b10001000) // opcode=0x8
// 	frame = append(frame, byte(len(message)))
// 	frame = append(frame, byte(statusCode>>8&0xFF))
// 	frame = append(frame, byte(statusCode&0xFF))
// 	frame = append(frame, message...)
// 	log.Println(frame)
// 	return frame
// }
