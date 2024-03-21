# v3/integrations/nrawsbedrock [![GoDoc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawsbedrock?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawsbedrock)

Package `nrawsbedrock` instruments https://github.com/aws/aws-sdk-go-v2/service/bedrockruntime requests.

This integration works independently of the `nrawssdk-v2` integration, which instruments AWS middleware components generally, while this one instruments Bedrock AI model invocations specifically and in detail.

```go
import "github.com/newrelic/go-agent/v3/integrations/nrawsbedrock"
```

For more information, see
[godocs](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawsbedrock).
