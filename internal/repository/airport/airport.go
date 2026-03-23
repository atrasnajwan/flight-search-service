package airport

import (
	"encoding/json"
	"os"
)

type Airport struct {
    data map[string]string
}

func NewInstance() *Airport {
    return &Airport{data: make(map[string]string)}
}

func (a *Airport) LoadFromJSON(path string) error {
    file, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(file, &a.data)
}

func (a *Airport) GetCity(code string) string {
    if city, ok := a.data[code]; ok {
        return city
    }
    return "Unknown"
}