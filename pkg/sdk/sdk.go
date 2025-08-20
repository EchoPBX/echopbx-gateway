package sdk

// Event es la estructura mínima que los plugins pueden publicar
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Bus es la interfaz pública del event bus
type Bus interface {
	Publish(ev Event)
}
