package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"io"
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
	go sendingMessage(c)
}

func receiveMessage(c net.Conn) {
	defer wg.Done()

	configurationBytesBuffer := make([]byte, 2)
	for {
		// Read first two configuration bytes
		_, err := io.ReadFull(c, configurationBytesBuffer)
		if err != nil {
			continue
		}

		// Determine payload length and mask
		payloadLength, mask, err := getPayloadLengthAndMask(configurationBytesBuffer, c)
		if err != nil {
			continue
		}

		// Read payload
		payload, err := readPayload(payloadLength, c)
		if err != nil {
			continue
		}

		// Decode payload using XOR encryption
		message := decodePayload(payload, mask)
		log.Println(message)
	}
}

func sendingMessage(c net.Conn) {
	defer wg.Done()

	inputReader := bufio.NewReader(os.Stdin)
	for {
		// Read message from stdin
		message, err := inputReader.ReadString('\n')
		if err != nil {
			continue
		}

		message = strings.TrimSpace(message)
		if len(message) < 1 {
			continue
		}

		// Determine payload length
		payloadLength := getPayloadLength(message)

		// Create frame and set first bytes (FIN + opcode)
		frame := createFrame(message, payloadLength)

		// Send frame to the client
		if _, err := c.Write(frame); err != nil {
			continue
		}
	}
}

// Receiving messages functions
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

// Sending messages functions
func getPayloadLength(message string) []byte {
	payloadLength := make([]byte, 0)
	if len(message) < 126 {
		payloadLength = append(payloadLength, byte(len(message)))
	} else if len(message) < 65536 {
		payloadLength = append(payloadLength, byte(126))
		for i := 0; i < 2; i++ {
			b := len(message) >> ((2 - i - 1) * 8) & 0xFF
			payloadLength = append(payloadLength, byte(b))
		}
	} else {
		payloadLength = append(payloadLength, byte(127))
		for i := 0; i < 8; i++ {
			b := len(message) >> ((2 - i - 1) * 8) & 0xFF
			payloadLength = append(payloadLength, byte(b))
		}
	}
	return payloadLength
}

func createFrame(message string, payloadLength []byte) []byte {
	frame := make([]byte, 0, 1+len(payloadLength)+len(message))
	frame = append(frame, 0b10000001)
	frame = append(frame, payloadLength...)
	return append(frame, message...)
}
