package nrotelhybrid

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func Test_isTransaction(t *testing.T) {
	validTraceID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	validSpanID := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		kind    oteltrace.SpanKind
		parent  oteltrace.SpanContext
		wantTxn bool
		wantWeb bool
		this    bool
		newRoot bool
		ctx     context.Context
	}{
		{
			name: "Remote parent exists and is valid. Kind is unspecified. New Root is false",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			wantTxn: true,
			wantWeb: false,
			newRoot: false,
		},
		{
			name: "Remote parent exists and is valid. Kind is unspecified. New Root is true",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			wantTxn: true,
			wantWeb: false,
			newRoot: true,
			this:    true,
		},
		{
			name: "Remote parent exists but is not valid. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: true,
			}),
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Remote parent exists. Kind is server.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name: "Remote parent exists. Kind is client.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindClient,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name: "Remote parent exists. Kind is consumer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Remote parent exists. Kind is producer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindProducer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is unspecified.",
			kind:    oteltrace.SpanKindUnspecified,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindUnspecified,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is internal.",
			kind:    oteltrace.SpanKindInternal,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is internal.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindInternal,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is server. txnChecker returns true.",
			kind:    oteltrace.SpanKindServer,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name:    "No parent. Kind is server. txnChecker returns false.",
			kind:    oteltrace.SpanKindServer,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns false.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name:    "No parent. Kind is client.",
			kind:    oteltrace.SpanKindClient,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name: "Parent is not remote. Kind is client.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindClient,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name:    "No parent. Kind is producer.",
			kind:    oteltrace.SpanKindProducer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is producer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindProducer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is consumer. txnCheck returns true.",
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is consumer. txnCheck returns false.",
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns false",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is not listed.",
			kind:    7,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is not listed.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    8,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is server. Ctx contains txn.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: false,
			wantWeb: true,
			ctx:     newrelic.NewContext(context.Background(), &newrelic.Transaction{}),
		},
		{
			name: "Parent is remote. Kind is server. Ctx contains txn.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: false,
			wantWeb: true,
			ctx:     newrelic.NewContext(context.Background(), &newrelic.Transaction{}),
		},
		{
			name: "Parent is not remote. Kind is consumer. Ctx contains txn.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: false,
			wantWeb: true,
			ctx:     newrelic.NewContext(context.Background(), &newrelic.Transaction{}),
		},
		{
			name: "Parent is not remote. Kind is client. Ctx contains txn.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindClient,
			wantTxn: false,
			wantWeb: true,
			ctx:     newrelic.NewContext(context.Background(), &newrelic.Transaction{}),
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns false. Ctx contains txn.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: false,
			wantWeb: true,
			ctx:     newrelic.NewContext(context.Background(), &newrelic.Transaction{}),
		},
	}
	for _, tt := range tests {
		if !tt.this {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			gotTxn, gotWeb := isTransaction(tt.kind, tt.parent, tt.newRoot, tt.ctx) // TODO CHANGE THIS
			if gotTxn != tt.wantTxn || gotWeb != tt.wantWeb {
				t.Errorf("isTransaction() = (%v, %v), want (%v, %v)", gotTxn, gotWeb, tt.wantTxn, tt.wantWeb)
			}
		})
	}
}

func Test_extractAttributeValue(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		val  attribute.Value
		want any
	}{
		{
			name: "Bool",
			val:  attribute.BoolValue(true),
			want: true,
		},
		{
			name: "Int64",
			val:  attribute.Int64Value(42),
			want: int64(42),
		},
		{
			name: "Float64",
			val:  attribute.Float64Value(3.14),
			want: 3.14,
		},
		{
			name: "String",
			val:  attribute.StringValue("hello"),
			want: "hello",
		},
		{
			name: "BoolSlice",
			val:  attribute.BoolSliceValue([]bool{true, false}),
			want: []bool{true, false},
		},
		{
			name: "Int64Slice",
			val:  attribute.Int64SliceValue([]int64{1, 2, 3}),
			want: []int64{1, 2, 3},
		},
		{
			name: "Float64Slice",
			val:  attribute.Float64SliceValue([]float64{1.1, 2.2}),
			want: []float64{1.1, 2.2},
		},
		{
			name: "StringSlice",
			val:  attribute.StringSliceValue([]string{"a", "b"}),
			want: []string{"a", "b"},
		},
		{
			name: "ByteSlice",
			val:  attribute.ByteSliceValue([]byte{0x01, 0x02}),
			want: []byte{0x01, 0x02},
		},
		{
			name: "Slice",
			val:  attribute.SliceValue(attribute.StringValue("a"), attribute.IntValue(1)),
			want: []attribute.Value{attribute.StringValue("a"), attribute.IntValue(1)},
		},
		{
			name: "Empty/invalid value",
			val:  attribute.Value{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAttributeValue(tt.val)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractAttributeValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nrotelhybridProcessor_switchSegmentType(t *testing.T) {

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		attributes     []attribute.KeyValue
		initialSegment nrSegment
		spanKind       oteltrace.SpanKind
		want           nrSegment
	}{
		{
			name:           "Segment is not a basic segment upon type cast. Should stay the same type.",
			initialSegment: &newrelic.DatastoreSegment{},
			spanKind:       oteltrace.SpanKindInternal,
			want:           &newrelic.DatastoreSegment{},
		},
		{
			name:           "Segment is not a basic segment (different type) upon type cast. Should stay the same type.",
			initialSegment: &newrelic.ExternalSegment{},
			spanKind:       oteltrace.SpanKindInternal,
			want:           &newrelic.ExternalSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is INTERNAL. Should stay the same type.",
			initialSegment: &newrelic.Segment{},
			spanKind:       oteltrace.SpanKindInternal,
			want:           &newrelic.Segment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is UNSPECIFIED. Should stay the same type.",
			initialSegment: &newrelic.Segment{},
			spanKind:       oteltrace.SpanKindUnspecified,
			want:           &newrelic.Segment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to External Segment.",
			initialSegment: &newrelic.Segment{},
			attributes:     []attribute.KeyValue{},
			spanKind:       oteltrace.SpanKindClient,
			want:           &newrelic.ExternalSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to External Segment. Attributes are present.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrURLFull), Value: attribute.StringValue("testURL")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.ExternalSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to Datastore Segment. Uses AttrDBSystem constant.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystem), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CLIENT.Should convert to Datastore Segment. Uses AttrDBSystemName constant.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystemName), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CLIENT.Should convert to Datastore Segment. Contains both DB constants.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystemName), Value: attribute.StringValue("test-system")},
				{Key: attribute.Key(AttrDBSystem), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is PRODUCER. Should convert to Producer Segment.",
			initialSegment: &newrelic.Segment{},
			spanKind:       oteltrace.SpanKindProducer,
			want:           &newrelic.MessageProducerSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is PRODUCER. Should convert to Producer Segment.  Has attributes for Messages.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrServerAddress), Value: attribute.StringValue("address")},
				{Key: attribute.Key(AttrServerPort), Value: attribute.StringValue("port")},
			},
			spanKind: oteltrace.SpanKindProducer,
			want:     &newrelic.MessageProducerSegment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CONSUMER. Should stay a basic segment.",
			initialSegment: &newrelic.Segment{},
			spanKind:       oteltrace.SpanKindConsumer,
			want:           &newrelic.Segment{},
		},
		{
			name:           "Segment is a basic segment upon type cast. SpanKind is CONSUMER. Should add attributes for Messages.",
			initialSegment: &newrelic.Segment{},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrServerAddress), Value: attribute.StringValue("address")},
				{Key: attribute.Key(AttrServerPort), Value: attribute.StringValue("port")},
			},
			spanKind: oteltrace.SpanKindConsumer,
			want:     &newrelic.Segment{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &hybridSegment{
				seg: tt.initialSegment,
			}
			p.switchSegmentType(tt.attributes, tt.spanKind)

			// if tt.want == nil {
			// 	if ok {
			// 		t.Errorf("Expected no segment, got segment at spanID = %v", tt.spanID)
			// 	}
			// 	return
			// }

			// if !ok {
			// 	t.Errorf("Expected segment at spanID = %v, got nothing", tt.spanID)
			// 	return
			// }

			if reflect.TypeOf(p.seg) != reflect.TypeOf(tt.want) {
				t.Errorf("Expected segment type of %v, got %v", reflect.TypeOf(p.seg), reflect.TypeOf(tt.want))
			}
		})
	}
}

func Test_checkMap(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		key     attribute.Key
		attrMap map[string]string
		want    string
		want2   bool
	}{
		{
			name:    "Nil map",
			key:     attribute.Key("some.key"),
			attrMap: nil,
			want:    "",
			want2:   false,
		},
		{
			name:    "Key present in map",
			key:     attribute.Key("some.key"),
			attrMap: map[string]string{"some.key": "nr.attribute"},
			want:    "nr.attribute",
			want2:   true,
		},
		{
			name:    "Key not present in map",
			key:     attribute.Key("missing.key"),
			attrMap: map[string]string{"some.key": "nr.attribute"},
			want:    "",
			want2:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := checkMap(tt.key, tt.attrMap)
			if got != tt.want {
				t.Errorf("checkMap() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("checkMap() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

type fakeSegment struct {
	attrs      map[string]any
	spanEvents []string
}

func (f *fakeSegment) End() {}

func (f *fakeSegment) AddAttribute(key string, val any) {
	if f.attrs == nil {
		f.attrs = map[string]any{}
	}
	f.attrs[key] = val
}

func (f *fakeSegment) AddLink(spanID string, traceID string, start time.Time) {}
func (f *fakeSegment) AddOtelSpanID(spanID string)                            {}
func (f *fakeSegment) AddSpanEventEvent(name string, start time.Time, attrs newrelic.SpanEventAttributes) {
	f.spanEvents = append(f.spanEvents, name)
}

func Test_nrotelhybridProcessor_addSegmentAttributes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		attributes []attribute.KeyValue
		attrMap    map[string]string
		want       map[string]interface{}
	}{
		{
			name:       "No attributes.",
			attributes: []attribute.KeyValue{},
			attrMap:    map[string]string{},
			want:       map[string]interface{}{},
		},
		{
			name: "Nil attrMap. Keys pass through unmapped.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.key", "value"),
			},
			attrMap: nil,
			want: map[string]interface{}{
				"otel.key": "value",
			},
		},
		{
			name: "Key present in attrMap is renamed.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.key", "value"),
			},
			attrMap: map[string]string{
				"otel.key": "nr.key",
			},
			want: map[string]interface{}{
				"nr.key": "value",
			},
		},
		{
			name: "Mix of mapped and unmapped keys, various value types.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.mapped", "value"),
				attribute.Int64("otel.unmapped", 42),
				attribute.Bool("otel.bool", true),
			},
			attrMap: map[string]string{
				"otel.mapped": "nr.mapped",
			},
			want: map[string]interface{}{
				"nr.mapped":     "value",
				"otel.unmapped": int64(42),
				"otel.bool":     true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &fakeSegment{}
			addSegmentAttributes(seg, tt.attributes, tt.attrMap)

			if seg.attrs == nil {
				seg.attrs = map[string]interface{}{}
			}
			if !reflect.DeepEqual(seg.attrs, tt.want) {
				t.Errorf("addSegmentAttributes() attrs = %v, want %v", seg.attrs, tt.want)
			}
		})
	}
}

func Test_addSegmentAttributes_DatastoreSegment(t *testing.T) {
	tests := []struct {
		name           string
		attributes     []attribute.KeyValue
		wantCollection string
		wantOperation  string
		wantQuery      string
		wantProduct    string
		wantOtherKV    struct {
			key   string
			value string
		}
	}{
		{
			name: "db.collection.name and db.operation.name set typed fields.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBCollectionName, "users"),
				attribute.String(AttrDBOperationName, "SELECT"),
			},
			wantCollection: "users",
			wantOperation:  "SELECT",
		},
		{
			name: "Legacy db.sql.table and db.operation set typed fields.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBSQLTable, "orders"),
				attribute.String(AttrDBOperation, "INSERT"),
			},
			wantCollection: "orders",
			wantOperation:  "INSERT",
		},
		{
			name: "db.statement sets ParameterizedQuery.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBStatement, "SELECT count(*) FROM pg_catalog.pg_tables"),
			},
			wantQuery: "SELECT count(*) FROM pg_catalog.pg_tables",
		},
		{
			name: "db.system sets Product",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBSystem, "Postgres"),
			},
			wantProduct: "Postgres",
		},
		{
			name: "db.system.name sets Product",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBSystemName, "Postgres"),
			},
			wantProduct: "Postgres",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &newrelic.DatastoreSegment{}
			addSegmentAttributes(seg, tt.attributes, OTELToNRDBAttributeMap)

			if seg.Collection != tt.wantCollection {
				t.Errorf("Collection = %v, want %v", seg.Collection, tt.wantCollection)
			}
			if seg.Operation != tt.wantOperation {
				t.Errorf("Operation = %v, want %v", seg.Operation, tt.wantOperation)
			}
			if seg.ParameterizedQuery != tt.wantQuery {
				t.Errorf("ParameterizedQuery = %v, want %v", seg.ParameterizedQuery, tt.wantQuery)
			}
			if seg.Product != newrelic.DatastoreProduct(tt.wantProduct) {
				t.Errorf("Product = %v, want %v", seg.Product, tt.wantProduct)
			}
		})
	}
}

func Test_addSegmentAttributes_MessageProducerSegment(t *testing.T) {
	tests := []struct {
		name         string
		attributes   []attribute.KeyValue
		wantLibrary  string
		wantDestName string
		wantDestType newrelic.MessageDestinationType
	}{
		{
			name: "messaging.system sets Library.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrMessagingSystem, "rabbitmq"),
			},
			wantLibrary: "rabbitmq",
		},
		{
			name: "messaging.destination.name sets DestinationName.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrMessagingDestinationName, "route-six-queue"),
			},
			wantDestName: "route-six-queue",
		},
		{
			name: "messaging.destination_kind sets DestinationType.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrMessagingDestinationKind, "queue"),
			},
			wantDestType: newrelic.MessageDestinationType("queue"),
		},
		{
			name: "messaging.operation sets DestinationType.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrMessagingOperation, "publish"),
			},
			wantDestType: newrelic.MessageDestinationType("publish"),
		},
		{
			name: "messaging.operation.type sets DestinationType.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrMessagingOperationType, "create"),
			},
			wantDestType: newrelic.MessageDestinationType("create"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &newrelic.MessageProducerSegment{}
			addSegmentAttributes(seg, tt.attributes, OTELToNRMessagingProducerAttributeMap)

			if seg.Library != tt.wantLibrary {
				t.Errorf("Library = %v, want %v", seg.Library, tt.wantLibrary)
			}
			if seg.DestinationName != tt.wantDestName {
				t.Errorf("DestinationName = %v, want %v", seg.DestinationName, tt.wantDestName)
			}
			if seg.DestinationType != tt.wantDestType {
				t.Errorf("DestinationType = %v, want %v", seg.DestinationType, tt.wantDestType)
			}
		})
	}
}

func Test_addSegmentAttributes_ExternalSegment(t *testing.T) {
	tests := []struct {
		name       string
		attributes []attribute.KeyValue
		wantURL    string
	}{
		{
			name: "url.full sets URL.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrURLFull, "http://example.com/path"),
			},
			wantURL: "http://example.com/path",
		},
		{
			name: "Legacy http.url sets URL.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrHTTPURL, "http://legacy.example.com/path"),
			},
			wantURL: "http://legacy.example.com/path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := &newrelic.ExternalSegment{}
			addSegmentAttributes(seg, tt.attributes, OTELToNRHTTPAttributeMap)

			if seg.URL != tt.wantURL {
				t.Errorf("URL = %v, want %v", seg.URL, tt.wantURL)
			}
		})
	}
}

func Test_otelEventAttributes_WriteAttributes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		eventAttributes    otelEventAttributes
		expectedAttrLength int
	}{
		{
			name:               "Empty otelEventAttributes",
			eventAttributes:    []attribute.KeyValue{},
			expectedAttrLength: 0,
		},
		{
			name: "Non-empty otelEventAttributes",
			eventAttributes: []attribute.KeyValue{
				{
					Key:   "Key1",
					Value: attribute.Value{},
				},
			},
			expectedAttrLength: 1,
		},
		{
			name: "Non-empty otelEventAttributes multiple",
			eventAttributes: []attribute.KeyValue{
				{
					Key:   "Key1",
					Value: attribute.Value{},
				},
				{
					Key:   "Key2",
					Value: attribute.Value{},
				},
			},
			expectedAttrLength: 2,
		},
		{
			name: "Non-empty otelEventAttributes multiple with same key",
			eventAttributes: []attribute.KeyValue{
				{
					Key:   "Key1",
					Value: attribute.Value{},
				},
				{
					Key:   "Key1",
					Value: attribute.Value{},
				},
			},
			expectedAttrLength: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			m := make(map[string]any)
			write := func(key string, val any) {
				m[key] = val
			}
			var a otelEventAttributes
			a = tt.eventAttributes
			a.WriteAttributes(write)
			if tt.expectedAttrLength != len(m) {
				t.Errorf("Expected attribute map of length %d, got attribute of length of length %d", tt.expectedAttrLength, len(m))
			}
		})
	}
}
