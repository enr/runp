package core

import (
	"testing"
)

func TestProcessEnv(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("empty env", func(t *testing.T) {
		vars := map[string]string{}
		preprocessor := newCliPreprocessor(vars)
		env := map[string]string{}

		result := processEnv(env, preprocessor)
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d items", len(result))
		}
	})

	t.Run("env with variables", func(t *testing.T) {
		vars := map[string]string{
			"test_var":  "value1",
			"other_var": "value2",
		}
		preprocessor := newCliPreprocessor(vars)
		env := map[string]string{
			"KEY1": "test-{{vars test_var}}-test",
			"KEY2": "simple-value",
		}

		result := processEnv(env, preprocessor)
		if len(result) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result))
		}
		if result["KEY1"] != "test-value1-test" {
			t.Errorf("Expected 'test-value1-test', got '%s'", result["KEY1"])
		}
		if result["KEY2"] != "simple-value" {
			t.Errorf("Expected 'simple-value', got '%s'", result["KEY2"])
		}
	})

	t.Run("env with multiple variables", func(t *testing.T) {
		vars := map[string]string{
			"test_var":  "value1",
			"other_var": "value2",
		}
		preprocessor := newCliPreprocessor(vars)
		env := map[string]string{
			"KEY1": "{{vars test_var}}",
			"KEY2": "{{vars other_var}}",
			"KEY3": "{{vars test_var}}-{{vars other_var}}",
		}

		result := processEnv(env, preprocessor)
		if len(result) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result))
		}
		if result["KEY1"] != "value1" {
			t.Errorf("Expected 'value1', got '%s'", result["KEY1"])
		}
		if result["KEY2"] != "value2" {
			t.Errorf("Expected 'value2', got '%s'", result["KEY2"])
		}
		if result["KEY3"] != "value1-value2" {
			t.Errorf("Expected 'value1-value2', got '%s'", result["KEY3"])
		}
	})
}
