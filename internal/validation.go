package internal

import "fmt"

func (m *Input) Validate(_ bool) error {
	if len(m.Value) < 5 {
		return fmt.Errorf("value is too short")
	}
	return nil
}
