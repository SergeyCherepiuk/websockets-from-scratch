package websockets

import (
	"errors"
	"io"
	"net"
)

type Frame struct {
	FIN     bool
	Opcode  byte
	Mask    [4]byte
	Payload []byte
}

func ReadFrame(c net.Conn) (Frame, error) {
	configurationBytes := make([]byte, 2)
	if _, err := io.ReadFull(c, configurationBytes); err != nil {
		return Frame{}, err
	}

	paylaodLength, err := getPayloadLength(c, configurationBytes[1])
	if err != nil {
		return Frame{}, err
	}

	mask := make([]byte, 4)
	if _, err := io.ReadFull(c, mask); err != nil {
		return Frame{}, err
	}

	payload := make([]byte, paylaodLength)
	if _, err := io.ReadFull(c, payload); err != nil {
		return Frame{}, err
	}

	frame := Frame{
		FIN:     getFIN(configurationBytes),
		Opcode:  getOpcode(configurationBytes),
		Mask:    [4]byte(mask),
		Payload: payload,
	}
	return frame, nil
}

func getPayloadLength(c net.Conn, secondByte byte) (uint64, error) {
	var sliceLength byte
	if secondByte-128 < 126 {
		return uint64(secondByte) - 128, nil
	} else if secondByte-128 == 126 {
		sliceLength = 2
	} else {
		sliceLength = 8
	}

	var payloadLength uint64
	payloadLengthSlice := make([]byte, sliceLength)
	if _, err := io.ReadFull(c, payloadLengthSlice); err != nil {
		return 0, err
	}
	for i, b := range payloadLengthSlice {
		payloadLength += uint64(b) << ((len(payloadLengthSlice) - i - 1) * 8)
	}
	return payloadLength, nil
}

func getFIN(configurationBytes []byte) bool {
	return (configurationBytes[0]>>7)&0b1 == 1
}

func getOpcode(configurationBytes []byte) byte {
	return configurationBytes[0] & 0b1111
}

func (frame Frame) Decode() (string, error) {
	if !frame.FIN || frame.Opcode == 0x0 {
		return "", errors.New("can't decode continuation frame")
	}

	message := make([]byte, 0, len(frame.Payload))
	for i := 0; i < len(frame.Payload); i++ {
		message = append(message, frame.Payload[i]^frame.Mask[i%4])
	}
	return string(message), nil
}
