package execution

import (
	"encoding/json"
	"fmt"
	"os"
)

// fixtureTransaction mirrors just the fields of backend/fixtures/demo_dataset.json
// that Execution needs — the scenario tag data-schema-agent added purely as
// fixture metadata (not a DB column, per the approved Component 1 scope).
type fixtureTransaction struct {
	ID       string  `json:"id"`
	Scenario *string `json:"scenario"`
}

type fixtureFile struct {
	Transactions []fixtureTransaction `json:"transactions"`
}

// LoadScenarios reads the fixture JSON at path and returns a map of
// transaction ID -> scenario tag, containing only transactions that have a
// non-null scenario. Read directly from the fixture rather than a DB column,
// per explicit instruction: this tag is demo-script metadata, not schema.
func LoadScenarios(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("execution: reading fixture %s: %w", path, err)
	}

	var f fixtureFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("execution: parsing fixture %s: %w", path, err)
	}

	scenarios := make(map[string]string)
	for _, txn := range f.Transactions {
		if txn.Scenario != nil && *txn.Scenario != "" {
			scenarios[txn.ID] = *txn.Scenario
		}
	}
	return scenarios, nil
}
