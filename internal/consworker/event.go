package consworker

type TaskName string

func (tn TaskName) String() string {
	return string(tn)
}

const (
	PublisherExternalEvent TaskName = "publisher_external_event"
)
