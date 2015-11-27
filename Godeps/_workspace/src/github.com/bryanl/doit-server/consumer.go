package doitserver

import "sync"

type Consumer struct {
	AccessToken string
	ID          string
	Err         string
	Message     string
}

type Consumers struct {
	mu        *sync.Mutex
	consumers map[string]chan Consumer
}

func NewConsumers() *Consumers {
	return &Consumers{
		mu:        &sync.Mutex{},
		consumers: map[string]chan Consumer{},
	}
}

func (cc *Consumers) Len() int {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	return len(cc.consumers)
}

func (cc *Consumers) Get(id string) chan Consumer {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, ok := cc.consumers[id]; !ok {
		c := make(chan Consumer)
		cc.consumers[id] = c
		return c
	}

	return cc.consumers[id]
}

func (cc *Consumers) Remove(id string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	delete(cc.consumers, id)
}
