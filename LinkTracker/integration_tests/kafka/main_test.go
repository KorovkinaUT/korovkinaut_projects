package kafkatest

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
)

var kafkaBrokers []string

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	brokers, terminate, err := helpers.StartKafka(ctx)
	cancel()

	if err != nil {
		fmt.Fprintf(os.Stderr, "start kafka: %v\n", err)
		os.Exit(1)
	}

	kafkaBrokers = brokers

	code := m.Run()

	terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := terminate(terminateCtx); err != nil {
		fmt.Fprintf(os.Stderr, "terminate kafka: %v\n", err)
	}
	terminateCancel()

	os.Exit(code)
}