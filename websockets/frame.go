package websockets

import (
	"errors"
	"io"
	"net"
)

type Frame struct {
	FIN      bool
	RSV1     bool
	RSV2     bool
	RSV3     bool
	Opcode   byte
	IsMasked bool
	Mask     []byte
	Payload  []byte
}

type UnmaskedFrame struct {
	message string
}

func (err UnmaskedFrame) Error() string {
	return err.message
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
		FIN:      getFIN(configurationBytes),
		RSV1:     getRSV1(configurationBytes),
		RSV2:     getRSV2(configurationBytes),
		RSV3:     getRSV3(configurationBytes),
		IsMasked: getIsMasked(configurationBytes),
		Opcode:   getOpcode(configurationBytes),
		Mask:     mask,
		Payload:  payload,
	}

	if !frame.IsMasked {
		return Frame{}, UnmaskedFrame{"Received frame isn't masked"}
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

func getRSV1(configurationBytes []byte) bool {
	return (configurationBytes[0]>>6)&0b1 == 1
}

func getRSV2(configurationBytes []byte) bool {
	return (configurationBytes[0]>>5)&0b1 == 1
}

func getRSV3(configurationBytes []byte) bool {
	return (configurationBytes[0]>>4)&0b1 == 1
}

func getOpcode(configurationBytes []byte) byte {
	return configurationBytes[0] & 0b1111
}

func getIsMasked(configurationBytes []byte) bool {
	return (configurationBytes[1]>>7)&0b1 == 1
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

func (frame Frame) Bytes() []byte {
	bytes := []byte{}

	var firstByte byte
	if frame.FIN {
		firstByte += 1 << 7
	}
	if frame.RSV1 {
		firstByte += 1 << 6
	}
	if frame.RSV2 {
		firstByte += 1 << 5
	}
	if frame.RSV3 {
		firstByte += 1 << 4
	}
	firstByte += frame.Opcode

	var secondByte byte
	if frame.IsMasked {
		secondByte += 1 << 7
	}
	var extendedPayloadLength byte
	if len(frame.Payload) < 126 {
		extendedPayloadLength = 0
		secondByte += byte(len(frame.Payload))
	} else if len(frame.Payload) < 65536 {
		extendedPayloadLength = 2
		secondByte += 126
	} else {
		extendedPayloadLength = 8
		secondByte += 127
	}
	extendedPayloadLengthSlice := make([]byte, extendedPayloadLength)
	for i := range extendedPayloadLengthSlice {
		extendedPayloadLengthSlice[i] = byte(len(frame.Payload) >> (int(extendedPayloadLength) - i - 1) & 0xFF)
	}

	bytes = append(bytes, firstByte, secondByte)
	bytes = append(bytes, extendedPayloadLengthSlice...)
	bytes = append(bytes, frame.Mask...)
	bytes = append(bytes, frame.Payload...)
	return bytes
}
