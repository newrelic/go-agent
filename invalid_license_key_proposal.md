# Proposal for new behavior: Invalid License Key

## Current Behavior

If the user configures the Application with a 40-digit license key, there's no
further validation at Application-creation time.
Then if they run the application, the behavior `StartTransaction()` is
indistinguishable from if they had set `enabled: false`. 

I suggest that a user of this library should have a way to distinguish between
a disabled app versus a misconfigured one.

## Proposal 1

A new public method on internal_app.app:

```
func (app *app) ValidateConfig() error
```

The implementation of this would check that the license key corresponds to an
actual NewRelic account, perhaps by hitting some 'ping' endpoint you might have
available, or by starting a demo transaction.

We wouldn't want to add it to the Application interface in a minor version
bump, because the interface is used as a return value from `NewApplication`
(rather than a concrete struct).
It could affect clients or their tests that use it.

## Proposal 2

Could this be checked as part of `newAppRun()`?

### PR vs. Issues

I'm filing this as a PR, but I'd rather have filed an issue. Is there a reason
issues aren't enabled for your repo?
