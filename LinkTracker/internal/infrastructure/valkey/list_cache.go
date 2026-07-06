package valkey

import (
	"context"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

type ListCache struct {
	client             valkey.Client
	ttl                time.Duration
	clientSideCacheTTL time.Duration
	timeout            time.Duration
}

func NewListCache(
	client valkey.Client,
	ttl time.Duration,
	clientSideCacheTTL time.Duration,
	timeout time.Duration,
) *ListCache {
	return &ListCache{
		client:             client,
		ttl:                ttl,
		clientSideCacheTTL: clientSideCacheTTL,
		timeout:            timeout,
	}
}

func (c *ListCache) Get(ctx context.Context, chatID int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	key := strconv.FormatInt(chatID, 10)

	result := c.client.DoCache(
		ctx,
		c.client.B().Get().Key(key).Cache(),
		c.clientSideCacheTTL,
	)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}

		return nil, err
	}

	value, err := result.AsBytes()
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *ListCache) Set(ctx context.Context, chatID int64, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	key := strconv.FormatInt(chatID, 10)

	return c.client.Do(
		ctx,
		c.client.B().
			Set().
			Key(key).
			Value(valkey.BinaryString(value)).
			Px(c.ttl).
			Build(),
	).Error()
}

func (c *ListCache) Delete(ctx context.Context, chatID int64) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.Do(
		ctx,
		c.client.B().
			Del().
			Key(strconv.FormatInt(chatID, 10)).
			Build(),
	).Error()
}
