package internal

const (
	// Methods used in collector communication.
	cmdMetrics      = "metric_data"
	cmdCustomEvents = "custom_event_data"
	cmdTxnEvents    = "analytic_event_data"
	cmdErrorEvents  = "error_event_data"
	cmdErrorData    = "error_data"
	cmdTxnTraces    = "transaction_sample_data"
	cmdSlowSQLs     = "sql_trace_data"
	cmdSpanEvents   = "span_event_data"
)
