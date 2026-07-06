package scrappertest

import (
	"context"
	"log"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
)

var testValkey *helpers.TestValkey

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testValkey, err = helpers.StartTestValkey(ctx)
	if err != nil {
		log.Printf("failed to start valkey: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testValkey.Terminate(ctx); err != nil {
		log.Printf("failed to terminate valkey: %v", err)
	}

	os.Exit(code)
}
