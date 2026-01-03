package task

import (
	"fmt"
)

// Returns a channel which posts updates about the print status.
func (task *Task) Subscribe() chan TaskStatus {
	task.Lock()
	defer task.Unlock()
	c := make(chan TaskStatus, 1)
	c <- task.status
	task.subscribers[fmt.Sprintf("%p", c)] = c
	return c
}

// Unsubscribe removes a previously subscribed channel.
func (task *Task) Unsubscribe(c chan TaskStatus) {
	task.Lock()
	defer task.Unlock()

	k := fmt.Sprintf("%p", c)
	_, ok := task.subscribers[k]
	if !ok {
		panic(fmt.Errorf("chan was not subscribed"))
	}
	delete(task.subscribers, k)
	close(c)
}

// broadcast sends the current status to all subscribers.
func (task *Task) broadcast() {
	task.RLock()
	defer task.RUnlock()

	taskStatus := task.status
	for _, c := range task.subscribers {
		c := c // shadow
		go func() {
			defer func() {
				// chan may be closed; ignored.
				recover()
			}()
			c <- taskStatus
			fmt.Printf("Broadcasted status to subscriber: %v\n", taskStatus)
		}()
	}
}
