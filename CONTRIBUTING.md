# Contributing

Contributions are always welcome. Before contributing please read the
[code of conduct](./CODE_OF_CONDUCT.md) and [search the issue tracker](../../issues); your issue may have already been discussed or fixed in `master`. To contribute,
[fork](https://help.github.com/articles/fork-a-repo/) this repository, commit your changes, and [send a Pull Request](https://help.github.com/articles/using-pull-requests/).

Note that our [code of conduct](./CODE_OF_CONDUCT.md) applies to all platforms and venues related to this project; please follow it in all your interactions with the project and its participants.

Please note, we only accept pull request for versions 3.8.0 of this project or greater.

## Version Support

When contributing, keep in mind that New Relic customers are running many different versions of Go, some of them pretty old.  For changes that depend on the newest version of Go, the tests will fail and the PR will be rejected.

Be aware that the instrumentation needs to work with a wide range of versions of the instrumented modules, and that code that looks nonsensical or overcomplicated may be that way for compatibility-related reasons.  Read all the comments and check the related tests before deciding whether existing code is incorrect.

If you’re planning on contributing a new feature or an otherwise complex contribution, we kindly ask you to start a conversation with the maintainer team by opening up a GitHub issue first. 

## Feature Requests

Feature requests should be submitted in the [Issue tracker](../../issues), with a description of the expected behavior & use case, where they’ll remain closed until sufficient interest, [e.g. :+1: reactions](https://help.github.com/articles/about-discussions-in-issues-and-pull-requests/), has been [shown by the community](../../issues?q=is%3Aissue+sort%3Areactions-%2B1-desc).
Before submitting an Issue, please search for similar ones in the
[closed issues](../../issues?q=is%3Aissue+is%3Aclosed).

## Pull Requests

Pull requests must pass all automated tests and must be reviewed by at least one maintaining engineer before being merged.

When contributing a new integration package, please follow the [Writing a New Integration Package](https://github.com/newrelic/go-agent/wiki/Writing-a-New-Integration-Package) wiki page.

## Contributor License Agreement

Keep in mind that when you submit your Pull Request, you'll need to sign the CLA via the click-through using CLA-Assistant. If you'd like to execute our corporate CLA, or if you have any questions, please drop us an email at opensource@newrelic.com.

For more information about CLAs, please check out Alex Russell’s excellent post,
[“Why Do I Need to Sign This?”](https://infrequently.org/2008/06/why-do-i-need-to-sign-this/).

## Slack

We host a public Slack with a dedicated channel for contributors and maintainers of open source projects hosted by New Relic.  If you are contributing to this project, you're welcome to request access to the #oss-contributors channel in the newrelicusers.slack.com workspace.  To request access, see https://newrelicusers-signup.herokuapp.com/.
