package domain

type Event struct {
	Name        string    `json:"name" bson:"name"`
	ServiceName string    `json:"service_name" bson:"service_name"`
	RepoURL     string    `json:"repo_url" bson:"repo_url"`
	TeamOwner   string    `json:"team_owner" bson:"team_owner"`
	Triggers    []Trigger `json:"triggers" bson:"triggers"`
}

type Trigger struct {
	ServiceName string            `json:"service_name" bson:"service_name"`
	Type        string            `json:"type" bson:"type"`
	Host        string            `json:"host" bson:"host"`
	Path        string            `json:"path" bson:"path"`
	Headers     map[string]string `json:"headers" bson:"headers"`
	Option      Opt               `json:"option" bson:"option"`
}

type Opt struct {
	MaxRetries int `json:"max_retries" bson:"max_retries"`
	Timeout    int `json:"timeout" bson:"timeout"`
	Retention  int `json:"retention" bson:"retention"`
	UniqueTTL  int `json:"unique_ttl" bson:"unique_ttl"`
}
