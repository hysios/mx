package viper

import (
	"os"
	"testing"

	"github.com/hysios/mx/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupTestViper(t *testing.T) (*viper.Viper, func()) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath(".")

	// Create a temporary config file
	content := []byte(`
foo: bar
nested:
  key: value
array:
  - item1
  - item2
`)
	err := os.WriteFile("config.yaml", content, 0644)
	assert.NoError(t, err)

	err = v.ReadInConfig()
	assert.NoError(t, err)

	cleanup := func() {
		os.Remove("config.yaml")
	}

	return v, cleanup
}

func TestNewViperProvider(t *testing.T) {
	v, cleanup := setupTestViper(t)
	defer cleanup()

	provider := NewViperProvider(v)
	assert.NotNil(t, provider)
	assert.Equal(t, v, provider.v)
}

func TestViperProvider_LookupPath(t *testing.T) {
	v, cleanup := setupTestViper(t)
	defer cleanup()

	provider := NewViperProvider(v)

	tests := []struct {
		name     string
		selector string
		want     interface{}
		wantOk   bool
	}{
		{"existing string", "foo", "bar", true},
		{"existing nested", "nested.key", "value", true},
		{"existing array", "array", []interface{}{"item1", "item2"}, true},
		{"non-existing", "nonexistent", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := provider.LookupPath(tt.selector)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.Equal(t, tt.want, got.Data())
			}
		})
	}
}

func TestViperProvider_Set(t *testing.T) {
	v, cleanup := setupTestViper(t)
	defer cleanup()

	provider := NewViperProvider(v)

	tests := []struct {
		name     string
		selector string
		value    interface{}
		want     interface{}
	}{
		{"set new value", "new_key", "new_value", nil},
		{"update existing value", "foo", "new_bar", "bar"},
		{"update nested value", "nested.key", "new_value", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old, err := provider.Set(tt.selector, tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, old)

			// Verify the new value is set
			got, ok := provider.LookupPath(tt.selector)
			assert.True(t, ok)
			assert.Equal(t, tt.value, got.Data())
		})
	}
}

func TestViperProvider_Update(t *testing.T) {
	v, cleanup := setupTestViper(t)
	defer cleanup()

	provider := NewViperProvider(v)

	update := map[string]interface{}{
		"foo":        "updated_bar",
		"new_key":    "new_value",
		"nested.key": "updated_value",
	}

	result := provider.Update(update)

	// Check if the update was successful
	for k, v := range update {
		got, ok := provider.LookupPath(k)
		t.Logf("key: %s, got: %v, ok: %v, %v", k, got, ok, v)
		assert.True(t, ok)
		assert.Equal(t, v, got.Data())
	}

	// Check if the result is correct
	assert.IsType(t, config.Map{}, result)
	// assert.True(t, reflect.DeepEqual(config.NewMap(v.AllSettings()), result))
}
