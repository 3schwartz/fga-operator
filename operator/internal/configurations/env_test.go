package configurations

import (
	"fmt"
	"os"
	"reflect"
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
	testCases := []struct {
		envValue       string
		expectedResult time.Duration
		description    string
	}{
		// Test case with no environment variable set (default value should be used)
		{"", 45 * time.Second, fmt.Sprintf("%s not set, expect default value of 45 seconds", RequeueAfterDuration)},

		// Test case with valid second values
		{"30s", 30 * time.Second, fmt.Sprintf("%s set to 30 seconds", RequeueAfterDuration)},
		{"90s", 90 * time.Second, fmt.Sprintf("%s set to 90 seconds", RequeueAfterDuration)},

		// Test case with valid minute values
		{"2m", 2 * time.Minute, fmt.Sprintf("%s set to 2 minutes", RequeueAfterDuration)},
		{"5m", 5 * time.Minute, fmt.Sprintf("%s set to 5 minutes", RequeueAfterDuration)},

		// Test case with valid hour values
		{"1h", 1 * time.Hour, fmt.Sprintf("%s set to 1 hour", RequeueAfterDuration)},
		{"3h", 3 * time.Hour, fmt.Sprintf("%s set to 3 hours", RequeueAfterDuration)},

		// Test case with invalid value (default should be used)
		{"invalid-value", 45 * time.Second, fmt.Sprintf("%s set to invalid value, expect default value of 45 seconds", RequeueAfterDuration)},
	}

	// Iterate over each test case
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			// Arrange
			err := os.Setenv(RequeueAfterDuration, testCase.envValue)
			if err != nil {
				t.Fatal(err)
			}
			logger := newTestLogger()

			// Act
			duration := GetRequeueAfterFromEnv(logger)

			// Assert
			if !reflect.DeepEqual(duration, testCase.expectedResult) {
				t.Errorf("expected %v, got %v", testCase.expectedResult, duration)
			}

			// Clean up environment variable
			err = os.Unsetenv(RequeueAfterDuration)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
