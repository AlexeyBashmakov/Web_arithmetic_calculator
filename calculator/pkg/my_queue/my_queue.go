package my_queue

import (
	"sync"
)

/*
Напишите потокобезопасную очередь ConcurrentQueue. Реализуете следующий интерфейс:

type Queue interface {
Enqueue(element interface{}) // положить элемент в очередь
Dequeue() interface{} // забрать первый элемент из очереди
}

Примечания
Код должен содержать следующую структуру:

type ConcurrentQueue struct {
queue []interface{} // здесь хранить элементы очереди
mutex sync.Mutex

По-сути, данная очередь - этой слайс (динамический массив) элементов типа interface - т.е. элементов,
которые могут хранить любой тип
}
*/
type ConcurrentQueue struct {
	queue []interface{} // здесь хранить элементы очереди
	mutex sync.Mutex
}

type Queue interface {
	Enqueue(element interface{}) // положить элемент в очередь
	Dequeue() interface{}        // забрать первый элемент из очереди
}

func (c *ConcurrentQueue) Enqueue(element interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.queue == nil {
		c.queue = make([]interface{}, 0)
	}
	c.queue = append(c.queue, element)
}

func (c *ConcurrentQueue) Dequeue() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.queue) == 0 {
		return nil
	}
	r := c.queue[0]
	c.queue = c.queue[1:]
	return r
}

func (c *ConcurrentQueue) Len() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.queue)
}
