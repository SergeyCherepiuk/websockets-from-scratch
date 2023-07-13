package main

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
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

	configurationBytesBuffer := make([]byte, 2)
	for {
		// Read first two configuration bytes
		_, err := io.ReadFull(c, configurationBytesBuffer)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		// Determine payload length and mask
		payloadLength, mask, err := getPayloadLengthAndMask(configurationBytesBuffer, c)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		// Read payload
		payload, err := readPayload(payloadLength, c)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		// Decode payload using XOR encryption
		message := decodePayload(payload, mask)
		log.Println(message)
	}
}

func getPayloadLengthAndMask(configurationBytesBuffer []byte, c net.Conn) (uint64, []byte, error) {
	secondByte := uint(configurationBytesBuffer[1])
	mask := make([]byte, 4)

	var payloadLength uint64
	if secondByte-128 < 126 {
		payloadLength = uint64(secondByte) - 128
	} else if secondByte-128 == 126 {
		payloadLengthSlice := make([]byte, 2)
		if _, err := io.ReadFull(c, payloadLengthSlice); err != nil {
			return 0, []byte{}, err
		}
		for i, b := range payloadLengthSlice {
			payloadLength += uint64(b) << ((len(payloadLengthSlice) - i - 1) * 8)
		}
	} else {
		payloadLengthSlice := make([]byte, 8)
		if _, err := io.ReadFull(c, payloadLengthSlice); err != nil {
			return 0, []byte{}, err
		}
		for i, b := range payloadLengthSlice {
			payloadLength += uint64(b) << ((len(payloadLengthSlice) - i - 1) * 8)
		}
	}
	if _, err := io.ReadFull(c, mask); err != nil {
		return 0, []byte{}, err
	}
	return payloadLength, mask, nil
}

func readPayload(payloadLength uint64, c net.Conn) (string, error) {
	payload := make([]byte, payloadLength)
	if _, err := io.ReadFull(c, payload); err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodePayload(payload string, mask []byte) string {
	message := make([]byte, len(payload))
	for i := 0; i < len(payload); i++ {
		message = append(message, payload[i]^mask[i%4])
	}
	return string(message)
}
