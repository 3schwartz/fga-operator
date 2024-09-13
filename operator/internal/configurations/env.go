package configurations

import (
	"fmt"
	"github.com/go-logr/logr"
	"os"
	"time"
)

const RequeueAfterDuration = "REQUEUE_AFTER_DURATION"

func GetRequeueAfterFromEnv(setupLog logr.Logger) time.Duration {
	defaultDuration := 45 * time.Second
	requeueAfterStr := os.Getenv(RequeueAfterDuration)

	if requeueAfterStr == "" {
		setupLog.Info(fmt.Sprintf("%s not set, using default", RequeueAfterDuration), "defaultDuration", defaultDuration)
		return defaultDuration
	}

	requeueAfter, err := time.ParseDuration(requeueAfterStr)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("Invalid %s value, using default", RequeueAfterDuration), "requeueAfterStr", requeueAfterStr, "defaultDuration", defaultDuration)
		return defaultDuration
	}

	setupLog.Info(fmt.Sprintf("Using %s from environment", RequeueAfterDuration), "requeueAfter", requeueAfter)
	return requeueAfter
}
