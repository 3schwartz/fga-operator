package configurations

import (
	"fmt"
	"github.com/go-logr/logr"
	"os"
	"time"
)

const ReconciliationInterval = "RECONCILIATION_INTERVAL"

func GetReconciliationInterval(setupLog logr.Logger) time.Duration {
	defaultDuration := 45 * time.Second
	reconciliationInterval := os.Getenv(ReconciliationInterval)

	if reconciliationInterval == "" {
		setupLog.Info(fmt.Sprintf("%s not set, using default", ReconciliationInterval), "defaultDuration", defaultDuration)
		return defaultDuration
	}

	requeueAfter, err := time.ParseDuration(reconciliationInterval)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("Invalid %s value, using default", ReconciliationInterval), "reconciliationInterval", reconciliationInterval, "defaultDuration", defaultDuration)
		return defaultDuration
	}

	setupLog.Info(fmt.Sprintf("Using %s from environment", ReconciliationInterval), "requeueAfter", requeueAfter)
	return requeueAfter
}
