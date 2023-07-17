package websockets

import "log"

func (conn Connection) transferMessages() {
	for {
		select {
		case <-conn.running:
			return
		default:
			frame, err := ReadFrame(conn.Conn)
			if err != nil {
				if err == ErrUnmaskedFrame {
					conn.sendCloseFrame(1002)
				}
				log.Println(err.Error())
				conn.Close()
			}

			if frame.Opcode == 0x8 {
				message, err := frame.Decode()
				if err != nil {
					log.Println(err.Error())
					conn.Close()
				}
				statusCode := int(message[0])<<8&0xFF00 + int(message[1])
				conn.sendCloseFrame(int(statusCode))
				conn.Close()
				continue
			}

			message, err := frame.Decode()
			if err != nil {
				log.Println(err.Error())
				conn.Close()
			}

			frameToClient := Frame{
				FIN:     true,
				Opcode:  frame.Opcode,
				Payload: []byte(message),
			}
			for _, c := range connections {
				if c != conn {
					c.Conn.Write(frameToClient.Bytes())
				}
			}
		}
	}
}
