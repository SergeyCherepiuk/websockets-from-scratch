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

	running := make(chan bool)

	wg.Add(1)
	go receiveMessage(c, running)

	wg.Add(1)
	go sendMessage(c, running)

	wg.Wait()
}

func receiveMessage(c net.Conn, running chan bool) {
	defer wg.Done()

	for {
		select {
		case <-running:
			return
		default:
			frame, err := ReadFrame(c)
			if err != nil {
				switch err.(type) {
				case *UnmaskedFrame:
					closeFrame := Frame{
						FIN:     true,
						Opcode:  0x8,
						Payload: []byte{0b11, 0b11101010}, // 1002
					}
					if _, err := c.Write(closeFrame.Bytes()); err != nil {
						log.Println(err.Error())
					}
					close(running)
				default:
					log.Println(err.Error())
					close(running)
				}
			}

			message, err := frame.Decode()
			if err != nil {
				log.Println(err.Error())
				close(running)
			}

			log.Println(message)
		}
	}
}

func sendMessage(c net.Conn, running chan bool) {
	defer wg.Done()

	userInput := make(chan string)
	go readUserInput(userInput, running)
	for {
		select {
		case <-running:
			return
		case message := <-userInput:
			frame := Frame{
				FIN:     true,
				Opcode:  0x1,
				Payload: []byte(message),
			}

			if _, err := c.Write(frame.Bytes()); err != nil {
				log.Println(err.Error())
				close(running)
			}
		}
	}
}

func readUserInput(userInputs chan<- string, running chan<- bool) {
	for {
		message, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println(err.Error())
			close(running)
		}
		userInputs <- strings.TrimSpace(message)
	}
}
