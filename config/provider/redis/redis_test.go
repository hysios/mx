package redis

import (
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisProvider(t *testing.T) {
	// TODO: Add test cases.
	db, mock := redismock.NewClientMock()

	mock.ExpectPing().SetVal("PONG")
	// mock.ExpectGet("test").RedisNil()

	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	_, ok := provider.load()
	assert.False(t, ok)
}

// LookupPath returns the value for the given selector.
func TestLookupPath(t *testing.T) {
	// TODO: Add test cases.
	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")
	mock.ExpectGet("test").SetVal(`{"a": 1}`)
	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	val, ok := provider.LookupPath("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val.Int())
}

// TestLookupPath of empty value
func TestLookupPathEmpty(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")
	mock.ExpectGet("test").RedisNil()

	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	_, ok := provider.LookupPath("a")
	assert.False(t, ok)
}

// Set sets the value for the given selector.
func TestSet(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")
	mock.ExpectGet("test").SetVal(`{"a": 1}`)
	mock.ExpectSet("test", `{"a": 1, "b": 2}`, 0).SetVal("OK")

	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	provider.Set("b", 2)
	val, ok := provider.LookupPath("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val.Int())
}

// TestSet of empty value
func TestSetEmpty(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")
	mock.ExpectGet("test").RedisNil()
	mock.ExpectSet("test", `{"b": 2}`, 0).SetVal("OK")

	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	provider.Set("b", 2)
	val, ok := provider.LookupPath("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val.Int())
}

// Update updates the values.
func TestUpdate(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")
	mock.ExpectGet("test").SetVal(`{"a": 1}`)
	mock.ExpectSet("test", `{"a": 1, "b": 2}`, 0).SetVal("OK")

	provider, err := NewRedisProvider(&RedisOption{
		Key:  "test",
		Mock: db,
	})

	assert.NoError(t, err)
	assert.NotNil(t, provider)

	old := provider.Update(map[string]interface{}{
		"b": 2,
	})
	val, ok := provider.LookupPath("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val.Int())

	var (
		oldm   = map[string]interface{}(old)
		expect = map[string]interface{}{"a": 1.0, "b": 2}
	)

	if !reflect.DeepEqual(expect, oldm) {
		t.Errorf("Update() = %v, want %v", oldm, expect)
	}
}
