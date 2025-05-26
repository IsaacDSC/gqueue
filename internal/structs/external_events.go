package structs

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type Data map[string]any

func (d Data) ToBytes() []byte {
	data, _ := json.Marshal(d)
	return data
}

type ExternalEvent struct {
	ID        uuid.UUID `bson:"id"`
	Name      string    `bson:"name"`
	Data      Data      `bson:"data"`
	Triggers  []Trigger `bson:"triggers"`
	Delivered []Trigger `bson:"delivered"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
	DeletedAt time.Time `bson:"deletedAt"`
}
