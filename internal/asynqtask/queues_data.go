package asynqtask

type Queue string

func (q Queue) String() string {
	return string(q)
}
