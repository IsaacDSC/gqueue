package pubadapter

import "github.com/hibiken/asynq"

type Opts struct {
	Attributes map[string]string
	AsynqOpts  []asynq.Option
	Type       string
}

var EmptyOpts = Opts{
	Attributes: make(map[string]string),
	AsynqOpts:  []asynq.Option{},
}
