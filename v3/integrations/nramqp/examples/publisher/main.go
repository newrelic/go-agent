package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nramqp"
	"github.com/newrelic/go-agent/v3/newrelic"

	amqp "github.com/rabbitmq/amqp091-go"
)

var indexHTML = `
<!DOCTYPE html>
<html>
<body>
	
<h1>Send a Rabbit MQ Message</h1>
	
<form>
	<label for="msg">Message:</label><br>
	  <input type="text" id="msg" name="msg"><br>
	<input type="submit" formaction="/message" value="Send">
</form>

</body>
</html>
	`

func failOnError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s\n", msg, err))
	}
}

type amqpServer struct {
	ch         *amqp.Channel
	exchange   string
	routingKey string
}

func NewServer(channel *amqp.Channel, exchangeName, routingKeyName string) *amqpServer {
	return &amqpServer{
		channel,
		exchangeName,
		routingKeyName,
	}
}

func (serv *amqpServer) index(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, indexHTML)
}

func (serv *amqpServer) publishPlainTxtMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// get the message from the HTTP form
	r.ParseForm()
	message := r.Form.Get("msg")

	err := nramqp.PublishWithContext(serv.ch,
		ctx,
		serv.exchange,   // exchange
		serv.routingKey, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})

	if err != nil {
		txn := newrelic.FromContext(ctx)
		txn.NoticeError(err)
	}

	serv.index(w, r)
}

// a rabit mq server must be running on localhost on port 5672
func main() {
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName("AMQP Publisher Example App"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigInfoLogger(os.Stdout),
	)

	if err != nil {
		panic(err)
	}

	nrApp.WaitForConnection(time.Second * 5)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	server := NewServer(ch, "", q.Name)

	http.HandleFunc(newrelic.WrapHandleFunc(nrApp, "/", server.index))
	http.HandleFunc(newrelic.WrapHandleFunc(nrApp, "/message", server.publishPlainTxtMessage))

	fmt.Println("\n\nlistening on: http://localhost:8000/")
	http.ListenAndServe(":8000", nil)

	nrApp.Shutdown(time.Second * 10)
}
