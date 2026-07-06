package valkeytest

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"
	valkeycache "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/valkey"
)

func TestListCache_Set_StoresExpectedJSONByChatIDKey(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(123)
	key := strconv.FormatInt(chatID, 10)
	expected := []byte(`{"links":[{"url":"https://github.com/test/repo","tags":["go","backend"]}]}`)

	t.Cleanup(func() {
		deleteKey(t, key)
	})

	//act
	err := testValkey.Cache.Set(ctx, chatID, expected)

	//assert
	if err != nil {
		t.Errorf("unexpected Set error: %v", err)
	}

	actual, found := getRawValue(t, key)
	if !found {
		t.Fatalf("expected value by key %q to be stored in Valkey", key)
	}

	if !bytes.Equal(expected, actual) {
		t.Errorf("expected raw JSON %s, got %s", string(expected), string(actual))
	}
}

func TestListCache_Get_MissingKeyReturnsNilNil(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(124)
	key := strconv.FormatInt(chatID, 10)

	deleteKey(t, key)

	//act
	actual, err := testValkey.Cache.Get(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected Get error: %v", err)
	}
	if actual != nil {
		t.Errorf("expected nil value for missing key, got %s", string(actual))
	}
}

func TestListCache_Get_ReturnsStoredJSON(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(125)
	key := strconv.FormatInt(chatID, 10)
	expected := []byte(`{"links":[{"url":"https://stackoverflow.com/questions/123","tags":["java"]}]}`)

	t.Cleanup(func() {
		deleteKey(t, key)
	})

	if err := testValkey.Cache.Set(ctx, chatID, expected); err != nil {
		t.Fatalf("failed to set cache value: %v", err)
	}

	//act
	actual, err := testValkey.Cache.Get(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected Get error: %v", err)
	}
	if !bytes.Equal(expected, actual) {
		t.Errorf("expected JSON %s, got %s", string(expected), string(actual))
	}
}

func TestListCache_Delete_RemovesValue(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(126)
	key := strconv.FormatInt(chatID, 10)
	value := []byte(`{"links":[{"url":"https://github.com/test/repo"}]}`)

	t.Cleanup(func() {
		deleteKey(t, key)
	})

	if err := testValkey.Cache.Set(ctx, chatID, value); err != nil {
		t.Fatalf("failed to set cache value: %v", err)
	}

	//act
	err := testValkey.Cache.Delete(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected Delete error: %v", err)
	}

	actual, err := testValkey.Cache.Get(ctx, chatID)
	if err != nil {
		t.Errorf("unexpected Get error after Delete: %v", err)
	}
	if actual != nil {
		t.Errorf("expected nil value after Delete, got %s", string(actual))
	}
}

func TestListCache_Set_ValueExpiresAfterTTL(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(127)
	key := strconv.FormatInt(chatID, 10)
	value := []byte(`{"links":[{"url":"https://github.com/test/repo"}]}`)

	shortTTLCache := valkeycache.NewListCache(
		testValkey.Client,
		100*time.Millisecond,
		testValkey.Config.Timeout,
		testValkey.Config.ClientSideCacheTTL,
	)

	t.Cleanup(func() {
		deleteKey(t, key)
	})

	if err := shortTTLCache.Set(ctx, chatID, value); err != nil {
		t.Fatalf("failed to set cache value: %v", err)
	}

	//act
	expired := eventually(3*time.Second, 50*time.Millisecond, func() bool {
		actual, err := shortTTLCache.Get(ctx, chatID)
		return err == nil && actual == nil
	})

	//assert
	if !expired {
		t.Errorf("expected key %q to expire after TTL", key)
	}
}

func getRawValue(t *testing.T, key string) ([]byte, bool) {
	t.Helper()

	result := testValkey.Client.Do(
		context.Background(),
		testValkey.Client.B().
			Get().
			Key(key).
			Build(),
	)

	if err := result.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, false
		}

		t.Fatalf("failed to get raw value from Valkey: %v", err)
	}

	value, err := result.AsBytes()
	if err != nil {
		t.Fatalf("failed to convert Valkey result to bytes: %v", err)
	}

	return value, true
}

func deleteKey(t *testing.T, key string) {
	t.Helper()

	err := testValkey.Client.Do(
		context.Background(),
		testValkey.Client.B().
			Del().
			Key(key).
			Build(),
	).Error()

	if err != nil {
		t.Errorf("failed to delete key %q: %v", key, err)
	}
}

func eventually(timeout time.Duration, interval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}

		time.Sleep(interval)
	}

	return condition()
}
