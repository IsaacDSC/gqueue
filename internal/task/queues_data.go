package task

type Queue string

func (q Queue) String() string {
	return string(q)
}

type Queues []Queue

func (q Queues) Contains(queue Queue) bool {
	for _, item := range q {
		if item == queue {
			return true
		}
	}
	return false
}
