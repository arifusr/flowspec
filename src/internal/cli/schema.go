package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/testing-cli/apitest/internal/schema"
)

// RunSchemaGenerate generates a sample JSON from a schema file.
func RunSchemaGenerate(path string) int {
	s, err := schema.LoadSchema(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 2
	}

	payload := schema.GeneratePayload(s)

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling: %s\n", err)
		return 2
	}

	fmt.Println(string(data))
	return 0
}
