package main

import (
	"crypto/sha1"
	"encoding/base64"
	"log"
	"net"
	"os"
)

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

	buffer := make([]byte, 1024)
	receiveChannel := make(chan []byte)
	go conn.receiveMessage(receiveChannel)
	for {
		n, err := c.Read(buffer)
		if err != nil {
			log.Println(err.Error())
			return
		}
		receiveChannel <- buffer[:n]
	}
}

func (conn Connection) receiveMessage(ch chan []byte) {
	for {
		buffer := <-ch
		payloadLength, maskStart := conn.getPayloadLengthAndMaskStart(buffer)
		message := conn.decodeFrame(buffer, payloadLength, maskStart)
		log.Println(message)
	}
}

func (conn Connection) getPayloadLengthAndMaskStart(buffer []byte) (uint64, byte) {
	var payloadLength uint64
	var maskStart byte
	if uint64(buffer[1])-128 < 126 {
		payloadLength = uint64(buffer[1]) - 128
		maskStart = 2
	} else if uint64(buffer[1])-128 == 126 {
		payloadLength += uint64(buffer[3]) << 8
		payloadLength += uint64(buffer[2])
		maskStart = 4
	} else {
		payloadLength += uint64(buffer[9]) << 56
		payloadLength += uint64(buffer[8]) << 48
		payloadLength += uint64(buffer[7]) << 40
		payloadLength += uint64(buffer[6]) << 32
		payloadLength += uint64(buffer[5]) << 24
		payloadLength += uint64(buffer[4]) << 16
		payloadLength += uint64(buffer[3]) << 8
		payloadLength += uint64(buffer[2])
		maskStart = 10
	}
	return payloadLength, maskStart
}

func (conn Connection) decodeFrame(buffer []byte, payloadLength uint64, maskStart byte) string {
	mask := buffer[maskStart : maskStart+4]
	message := make([]byte, payloadLength)
	for i := 0; i < int(payloadLength); i++ {
		char := buffer[i+int(maskStart)+4] ^ mask[i%4]
		message = append(message, char)
	}
	return string(message)
}
