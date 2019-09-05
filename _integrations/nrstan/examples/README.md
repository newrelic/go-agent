# Example STAN app
In this example app you can find several different ways of instrumenting streaming STAN functions using New Relic. In order to run the app, make sure the following assumptions are correct: 
* Your New Relic license key is available as an environment variable named `NEW_RELIC_LICENSE_KEY`
* A STAN server is running locally at the `nats.DefaultURL`, using the cluster id `test-cluster`