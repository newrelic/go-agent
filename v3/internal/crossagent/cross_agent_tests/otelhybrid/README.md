# HybridAgentCrossAgentTestExample

Hybrid agent cross agent tests with an example implementation showing how to interpret them. The [TestCaseDefinitions.json](./TestCaseDefinitions.json) file defines the tests that should be run. Each test case describes a simulated run of an application along with validations that should occur at different points in time during the running of the simulated application and/or after the agent harvest cycle.

## How to interpret the test cases.

At the top level, the test cases file contains an array of test case definitions. Each test case definition has the following structure.

* `testDescription` field - Contains a brief description of the purpose of the test.
* `operations` field - Contains a tree structure that describes the things to run within the simulated application.
* `agentOutput` field (optional) - Contains a collection of different agent payloads to verify.

### Operations

Operations represent the actions to take within a simulated run of an application. An operation can contain simulated application logic, New Relic or OpenTelemetry api interactions, and/or things to validate during the operation. Operations have the following structure.

* `command` field - The name of the simulated action to take. There is a fixed set of commands that can be run and each command has its own rules for evaluating the parameters collection.
* `parameters` field - This field is meant to be interpreted as a collection of key value pairs that are specific to the command that should be run.
* `childOperations` field - The collection of `operations` that should be run before the current operation can complete running. The operation should run before the child operations are run, but the operation should not complete untile the child operations are all complete. This allows us to define a tree structure representing a simulated application run.
* `assertions` field - The collection of validation rules to run during an operation. These assertions should be performed after an operation's child operations are completed but before the current operation is completed.

#### Commands

* `DoWorkInSpan` - Supports two parameters `spanKind` that corresponds to the [kind](https://opentelemetry.io/docs/specs/otel/trace/api/#spankind) of the span to be created using the OpenTelemetry API, and `spanName` which defines the name of the span. This command creates and starts an OpenTelemetry span before running any child operations or assertions. The created span should end when this operation completes.
* `DoWorkInSpanWithRemoteParent` - Supports two parameters `spanKind` that corresponds to the [kind](https://opentelemetry.io/docs/specs/otel/trace/api/#spankind) of the span to be created using the OpenTelemetry API, and `spanName` which defines the name of the span. This command creates and starts an OpenTelemetry span using a ["remote" SpanContext](https://opentelemetry.io/docs/specs/otel/trace/api/#spancontext) before running any child operations or assertions. The created span should end when this operation completes.
* `DoWorkInSpanWithInboundContext` - Supports 5 parameters. `spanKind` that corresponds to the [kind](https://opentelemetry.io/docs/specs/otel/trace/api/#spankind) of the span to be created using the OpenTelemetry API, `spanName` which defines the name of the span, `traceIdInHeader` which is used to populate the traceId component of the `traceparent` W3C trace context header, `spanIdInHeader` which is used to populate the spanId component of the `traceparent` W3C trace context header, and `sampledFlagInHeader` which is used to populate the sampled flag component of the `traceparent` W3C trace context header and the sampled flag component of the New Relic data model stored in the `tracestate` W3C trace context header. Other than the sampled flag, the `tracestate` header value should be populated with values to allow the trace to be accepted (such as having an appropriate trusted account id). This command creates and starts an OpenTelemetry span using a simulated inbound context (containing `traceparent` and `tracestate` headers) and the [OpenTelemetry extract headers API](https://opentelemetry.io/docs/specs/otel/context/api-propagators/#extract), before running any child operations or assertions. The created span should end when this operation completes.
* `DoWorkInTransaction` - Supports a single parameter `transactionName` that corresponds to the name of the transaction to create using the New Relic agent API to create and start a transaction. This command should create a transaction before running any child operations and assertions. The transaction should end when this operation completes.
* `DoWorkInSegment` - Supports a single parameter `segmentName` that corresponds to the name of the segment that should be created using the New Relic agent API for creating segments. The segment should be created and started before running any child operations and assertions. The segment should end when this operation completes.
* `AddOTelAttribute` - Supports two parameters `name` and `value` that correspond to the name and value (respectively) that should be added to the current span using the OpenTelemetry API. This requires getting the current span using the OpenTelemetry API and adding the attribute to that span using the OpenTelemetry API.
* `RecordExceptionOnSpan` - Supports a single parameter `errorMessage` which contains the error message for the exception that should be recorded on the current span using the OpenTelemetry [RecordException API](https://opentelemetry.io/docs/specs/otel/trace/api/#record-exception). This operation requires getting the current span using the OpenTelemetry API and calling the `RecordException` API on that span. The exception should be recorded before running an child operations and assertions.
* `SimulateExternalCall` - Supports a single parameter `url` containing the url of the simulated external call. This command needs to create a request header collection that other child commands can use to inject distributed tracing headers into, and allow assertions to validate the request headers.
* `OTelInjectHeaders` - This command has no parameters, and understands how to interact with the current simulated external call to use the [OpenTelemetry header injection API](https://opentelemetry.io/docs/specs/otel/context/api-propagators/#inject) to insert the W3C trace context headers.
* `NRInjectHeaders` - This command has no parameters, and understands how to interact with the current simulated external call to use the New Relic distributed tracing API to insert the W3C trace context headers.

#### Assertions

Each assertion contains the following information.

* `description` field - Contains an explanation of what is being validated.
* `rule` field - Contains object that defines the validation rule. Each validation rule includes an `operator` field containing the name of the validation rule and a `parameters` collection containing the things that the `operator` should act on.

##### Rules

Each rule has their own set of parameters.

The `NotValid` rule has a single parameter named `object`. The object parameter contains a string that describes which `object` should be checked to determine if it is valid. Two of the supportted objects are `currentOTelSpan` and `currentTransaction` which represent the current OpenTelemetry span and current New Relic transaction respectively. The logic for determining the validity of these objects will vary from language. In some cases `null`/`nil` values represent that the object is not valid. In other cases, there is a no-op instance of an object that means the object is not valid.

The `Equals` rule represents an equality comparison between the objects represented by the `left` and `right` parameters. Here are the comparable objects.

* `currentOTelSpan.traceId` - The trace id defined on the current OpenTelemetry span.
* `currentTransaction.traceId` - The trace id defined on the current New Relic transaction.
* `currentOTelSpan.spanId` - The span id defined on the current OpenTelemetry span.
* `currentSegment.spanId` - The span id defined on the current New Relic segment (or the equivalent New Relic span depending on the agent's data model).
* `currentTransaction.sampled` - The sampled flag on the current New Relic transaction.
* `injected.traceId` - The trace id header value extracted from the simulated external call.
* `injected.spanId` - The span id header value extracted from the simulated external call.
* `injected.sampled` - The sampled flag header value extracted from the simulated external call.

The `Matches` rule represents a case-insensitive string comparsion between an object and a string constant. The object can be any of the objects that represent a string that is also supported by the `Equals` rule.

### Agent Output

After running all of the operations in a test, the test runner will need to either wait for or trigger the necessary harvests so that we can validate that the agent produced the expected telemetry data.

* If a telemetry type is not mentioned, then it should not be validated.
* If a telemetry type is mentioned but its collection is empty, `[]`, it means that we should validate that no data was collected for that telemetry type. This may mean that the harvest does not happen for that telemetry type.

#### Transactions

The `transactions` collection refers to transaction events. The framework currently only supports checking for the existance of a transaction by name. This name does not need to be exact match, because by default New Relic transaction names have different `/` delimited parts of the transaction name.

#### Spans

The `spans` collection refers to the span events generated by the agent. The main difference between the fields on this object and the span event is that the attribute collection is combined into a single collection. The test cases do not need to differentiate between intrinsic, agent, and user/custom attribute collections. If an attribute is found in any of those collections, then it should be used.