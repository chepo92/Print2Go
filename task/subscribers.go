package task

import (
	"fmt"
)

// Returns a channel which posts updates about the print status.
func (t *Task) Subscribe() chan TaskStatus {
	t.Lock()
	defer t.Unlock()
	c := make(chan TaskStatus, 1)
	c <- t.status
	t.subscribers[fmt.Sprintf("%p", c)] = c
	return c
}

func (t *Task) Unsubscribe(c chan TaskStatus) {
	t.Lock()
	defer t.Unlock()

	k := fmt.Sprintf("%p", c)
	_, ok := t.subscribers[k]
	if !ok {
		panic(fmt.Errorf("chan was not subscribed"))
	}
	delete(t.subscribers, k)
	close(c)
}

func (t *Task) broadcast() {
	t.RLock()
	defer t.RUnlock()

	ts := t.status
	for _, c := range t.subscribers {
		c := c // shadow
		go func() {
			defer func() {
				// chan may be closed; ignored.
				recover()
			}()
			c <- ts
		}()
	}
}
