package settings

import (
	"encoding/json"
	"io"
	"os"
)

func Load(config any) {
	file, err := os.Open("settings.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(byteValue, &config); err != nil {
		panic(err)
	}
}
