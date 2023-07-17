package websockets

import "errors"

type FrameQueue []Frame

func (queue *FrameQueue) Enqueue(frame Frame) {
	*queue = append(*queue, frame)
}

func (queue *FrameQueue) Dequeue() (Frame, error) {
	if len(*queue) < 1 {
		return Frame{}, errors.New("queue is empty")
	}
	frame := (*queue)[0]
	*queue = (*queue)[1:]
	return frame, nil
}
