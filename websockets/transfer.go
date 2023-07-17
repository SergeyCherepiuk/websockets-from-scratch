package websockets

import (
	"errors"
	"log"
	"reflect"
	"strings"
)

func (conn Connection) transferMessages() {
	for {
		select {
		case <-conn.running:
			return
		default:
			frame, err := ReadFrame(conn.Conn)
			if err != nil {
				log.Println(err.Error())
				conn.Close()
			}
			conn.queue.Enqueue(frame)

			if !frame.IsMasked {
				log.Println(errors.New("frame isn't masked").Error())
				conn.Close()
			}

			if frame.Opcode == 0x8 {
				message, err := frame.Decode()
				if err != nil {
					log.Println(err.Error())
					conn.Close()
				}
				statusCode := int(message[0])<<8&0xFF00 + int(message[1])
				conn.sendCloseFrame(statusCode)
				conn.Close()
				continue
			}

			var opcode byte
			var message string
			if frame.FIN {
				opcode = conn.queue[0].Opcode
				messageParts := []string{}
				for {
					continuationFrame, err := conn.queue.Dequeue()
					if err != nil {
						break
					}
					continuationFrame.FIN = true
					continuationFrame.Opcode = opcode
					messagePart, err := continuationFrame.Decode()
					if err != nil {
						log.Println(err.Error())
						conn.Close()
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
			log.Println(frameToClient)
			for _, c := range connections {
				if !reflect.DeepEqual(c, conn) {
					c.Conn.Write(frameToClient.Bytes())
				}
			}
		}
	}
}
