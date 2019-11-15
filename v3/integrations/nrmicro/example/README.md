# Example Go Micro apps
In this directory you will find several example Go Micro apps that are instrumented using the New Relic agent. All of the apps assume that your New Relic license key is available as an environment variable named `NEW_RELIC_LICENSE_KEY`

They can be run the standard way:
* The sample Pub/Sub app: `go run pubsub/main.go` instruments both a publish and a subscribe method
* The sample Server app: `go run server/server.go` instruments a handler method
* The sample Client app: `go run client/client.go` instruments the client.  
  * Note that in order for this to function, the server app must also be running.
 