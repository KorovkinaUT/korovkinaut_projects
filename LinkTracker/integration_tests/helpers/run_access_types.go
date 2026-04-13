package helpers

import "testing"

var AccessTypes = []string{"SQL", "GOQU"}

func RunAllAccessTypes(t *testing.T, testFn func(t *testing.T, accessType string)) {
	t.Helper()

	for _, accessType := range AccessTypes {
		accessType := accessType

		t.Run(accessType, func(t *testing.T) {
			testFn(t, accessType)
		})
	}
}