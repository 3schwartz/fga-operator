package configurations

import (
	"fmt"
	"github.com/go-logr/logr"
	"os"
	"time"
)

const ReconciliationInterval = "RECONCILIATION_INTERVAL"
const DefaultReconciliationInterval = 10 * time.Second

func GetReconciliationInterval(setupLog logr.Logger) time.Duration {
	reconciliationInterval := os.Getenv(ReconciliationInterval)

	if reconciliationInterval == "" {
		setupLog.Info(fmt.Sprintf("%s not set, using default", ReconciliationInterval), "defaultDuration", DefaultReconciliationInterval)
		return DefaultReconciliationInterval
	}

	requeueAfter, err := time.ParseDuration(reconciliationInterval)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("Invalid %s value, using default", ReconciliationInterval), "reconciliationInterval", reconciliationInterval, "defaultDuration", DefaultReconciliationInterval)
		return DefaultReconciliationInterval
	}

	setupLog.Info(fmt.Sprintf("Using %s from environment", ReconciliationInterval), "requeueAfter", requeueAfter)
	return requeueAfter
}
