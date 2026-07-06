package helpers

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	valkeyclient "github.com/valkey-io/valkey-go"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	valkeycache "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/valkey"
)

const valkeyImage = "valkey/valkey:8.1.3"

type TestValkey struct {
	Container testcontainers.Container
	Client    valkeyclient.Client
	Cache     service.ListCache
	Config    *config.ValkeyConfig
	Address   string
}

func StartTestValkey(ctx context.Context) (*TestValkey, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        valkeyImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	address := net.JoinHostPort(host, port.Port())

	cfg := &config.ValkeyConfig{
		Addresses:                []string{address},
		TTL:                      24 * time.Hour,
		Timeout:                  5 * time.Second,
		ClientSideCachingEnabled: false,
		ClientSideCacheTTL:       time.Minute,
	}

	client, err := valkeycache.NewClient(cfg)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	cache := valkeycache.NewListCache(
		client,
		cfg.TTL,
		cfg.Timeout,
		cfg.ClientSideCacheTTL,
	)

	return &TestValkey{
		Container: container,
		Client:    client,
		Cache:     cache,
		Config:    cfg,
		Address:   address,
	}, nil
}

func NewTestValkey(t *testing.T) *TestValkey {
	t.Helper()

	ctx := context.Background()

	valkey, err := StartTestValkey(ctx)
	if err != nil {
		t.Fatalf("failed to start valkey: %v", err)
	}

	t.Cleanup(func() {
		if err := valkey.Terminate(context.Background()); err != nil {
			t.Errorf("failed to terminate valkey: %v", err)
		}
	})

	return valkey
}

func (v *TestValkey) Terminate(ctx context.Context) error {
	if v.Client != nil {
		v.Client.Close()
	}

	if v.Container == nil {
		return nil
	}

	return v.Container.Terminate(ctx)
}
