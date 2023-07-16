package websockets

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"log"
	"net"
	"os"
	"strings"
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

	wg.Add(1)
	go sendMessage(c)
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

func sendMessage(c net.Conn) {
	defer wg.Done()

	for {
		message, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println(err.Error())
			return
		}
		message = strings.TrimSpace(message)

		frame := Frame{
			FIN:     true,
			Opcode:  0x1,
			Payload: []byte(message),
		}

		if _, err := c.Write(frame.Bytes()); err != nil {
			log.Println(err.Error())
			return
		}
	}
}
