package configurations

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

func newTestLogger() logr.Logger {
	return funcr.New(func(prefix, args string) {
	}, funcr.Options{})
}

func TestGetRequeueAfterFromEnv(t *testing.T) {
	// Define test cases with different environment values and expected outcomes
	testCases := []struct {
		envValue       string
		expectedResult time.Duration
		description    string
	}{
		// Test case with no environment variable set (default value should be used)
		{"", 45 * time.Second, "REQUEUE_AFTER_DURATION not set, expect default value of 45 seconds"},

		// Test case with valid second values
		{"30s", 30 * time.Second, "REQUEUE_AFTER_DURATION set to 30 seconds"},
		{"90s", 90 * time.Second, "REQUEUE_AFTER_DURATION set to 90 seconds"},

		// Test case with valid minute values
		{"2m", 2 * time.Minute, "REQUEUE_AFTER_DURATION set to 2 minutes"},
		{"5m", 5 * time.Minute, "REQUEUE_AFTER_DURATION set to 5 minutes"},

		// Test case with valid hour values
		{"1h", 1 * time.Hour, "REQUEUE_AFTER_DURATION set to 1 hour"},
		{"3h", 3 * time.Hour, "REQUEUE_AFTER_DURATION set to 3 hours"},

		// Test case with invalid value (default should be used)
		{"invalid-value", 45 * time.Second, "REQUEUE_AFTER set to invalid value, expect default value of 45 seconds"},
	}

	// Iterate over each test case
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			// Arrange
			err := os.Setenv("REQUEUE_AFTER", testCase.envValue)
			if err != nil {
				t.Fatal(err)
			}
			logger := newTestLogger()

			// Act
			duration := GetRequeueAfterFromEnv(logger)

			// Assert
			assert.Equal(t, testCase.expectedResult, duration, testCase.description)

			// Clean up environment variable
			err = os.Unsetenv("REQUEUE_AFTER")
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
