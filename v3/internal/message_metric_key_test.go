package internal

import "testing"

func TestNameWithConsumerMessageMetricKey(t *testing.T) {

	consumer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        true,
		DestinationName: "DestinationName",
		DestinationTemp: false,
	}

	expect := "Message/" + consumer.Library + "/" + consumer.DestinationType + "/" + "Named" + "/" + consumer.DestinationName
	actual := MessageMetricKey.Name(consumer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}

func TestNameWithConsumerMessageMetricKeyWithDestinationTemp(t *testing.T) {

	consumer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        true,
		DestinationName: "DestinationName",
		DestinationTemp: true,
	}

	expect := "Message/" + consumer.Library + "/" + consumer.DestinationType + "/" + "Temp"
	actual := MessageMetricKey.Name(consumer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}

func TestNameWithConsumerMessageMetricKeyWithEmptyDestinationName(t *testing.T) {

	consumer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        true,
		DestinationName: "",
		DestinationTemp: false,
	}

	expect := "Message/" + consumer.Library + "/" + consumer.DestinationType + "/" + "Named/Unknown"
	actual := MessageMetricKey.Name(consumer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}

func TestNameWithProducerMessageMetricKey(t *testing.T) {

	producer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        false,
		DestinationName: "DestinationName",
		DestinationTemp: true,
	}

	expect := "MessageBroker/" + producer.Library + "/" + producer.DestinationType + "/" + "Produce" + "/" + "Temp"
	actual := MessageMetricKey.Name(producer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}

func TestNameWithProducerMessageMetricKeyWithDestinationName(t *testing.T) {

	producer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        false,
		DestinationName: "DestinationName",
		DestinationTemp: false,
	}

	expect := "MessageBroker/" + producer.Library + "/" + producer.DestinationType + "/" + "Produce" + "/" + "Named" + "/" + producer.DestinationName
	actual := MessageMetricKey.Name(producer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}

func TestNameWithProducerMessageMetricKeyWithEmptyDestinationName(t *testing.T) {

	producer := MessageMetricKey{
		Library:         "hello",
		DestinationType: "DestinationType",
		Consumer:        false,
		DestinationName: "",
		DestinationTemp: false,
	}

	expect := "MessageBroker/" + producer.Library + "/" + producer.DestinationType + "/" + "Produce" + "/" + "Named" + "/" + "Unknown"
	actual := MessageMetricKey.Name(producer)

	if expect != actual {
		t.Errorf("got %v, want %v", actual, expect)
	}
}
