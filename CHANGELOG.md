## ChangeLog

* Support for queue time added:  if the inbound request contains an
  "X-Request-Start" or "X-Queue-Start" header with a unix timestamp, the agent
  will report queue time metrics.  Queue time will appear on the application
  overview chart.  The timestamp may fractional seconds, milliseconds, or
  microseconds: the agent will deduce the correct units.
