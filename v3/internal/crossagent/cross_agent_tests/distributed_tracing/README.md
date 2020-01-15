### Trace Context test details

The Trace Context test cases in `trace_context.json` are meant to be used to verify the
creation and forwarding of W3C Trace Context headers within the agent and the attributes
and metrics that get created during that process.

Each test case should correspond to a simulated inbound header or creation of a header in
the agent under test. Here's what the various fields in each test case mean:

| Name | Meaning |
| ---- | ------- |
| `test_name` | A human-meaningful name for the test case. |
| `trusted_account_key` | The account ids the agent can trust. |
| `account_id` | The account id the agent would receive on connect. |
| `web_transaction` | Whether the transaction that's tested is a web transaction or not. |
| `raises_exception` | Whether to simulate an exception happening within the transaction or not, resulting in a transaction error event. |
| `force_sampled_true` | Whether to force a transaction to be sampled or not. |
| `transport_type` | The transport type for the inbound request. |
| `inbound_headers` | The headers you should mock coming into the agent. |
| `outbound_payloads` | The exact/expected/unexpected values for outbound headers. |
| `intrinsics` | The exact/expected/unexpected attributes for events. |
| `expected_metrics` | The expected metrics and associated counts as a result of the test. |
| `span_events_enabled` | Whether span events are enabled in the agent or not. |

The `outbound_payloads` and `intrinsics` field can have nested values, for example:
```javascript
...
    "intrinsics": {
       "target_events": ["Transaction", "Span"],
       "common":{
         "exact": {
           "traceId": "da8bc8cc6d062849b0efcf3c169afb5a"
         },
         "expected": ["guid"],
         "unexpected": ["grandparentId"]
       },
       "Transaction": {
         "exact": {
           "parent.type": "App",
           "parent.app": "2827902",
           "parent.account": "33",
           "parent.transportType": "HTTP",
           "parentId": "e8b91a159289ff74",
           "parentSpanId": "7d3efb1b173fecfa"
         },
         "expected": ["parent.transportDuration"]
       },
       "Span": {
         "exact": {
           "parentId": "7d3efb1b173fecfa",
           "trustedParentId": "7d3efb1b173fecfa",
           "tracingVendors": ""
         },
         "expected": ["transactionId"],
         "unexpected": ["parent.transportDuration", "parent.type", "parent.app", "parent.account", "parent.transportType"]
       }
     },
    ...
```

`target_events` is paired with the `common` block. So anything in the common block should be checked for any event type in the
`target_events` list. So for instance, this test should check that both the Transaction and Span events
have a `guid`, both have `da8bc8cc6d062849b0efcf3c169afb5a` as the `traceId`, and both don't have a `grandparentId` attribute.
The `Transaction` block means anything in there should only apply to the transaction object. Same for the `Span` block.

The same idea goes for the `outbound_payloads` block but will apply specifically for the outbound `traceparent` header and `tracestate` header.
