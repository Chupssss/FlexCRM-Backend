package pipelines

import "github.com/google/uuid"

type PipelineOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
