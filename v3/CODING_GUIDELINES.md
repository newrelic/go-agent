# Go Coding Guidelines and Best Practices

These guidelines apply to all Go code in this repository (`v3/newrelic`, `v3/internal`, `v3/integrations/*`, `v3/examples`). When modifying existing code that does not conform, see "Handling Non-Conforming Legacy Code" at the bottom.

## Agent Patterns

To maintain consistency in our implementation of the agent code, follow these patterns when adding or maintaining features (more will be added over time).

### Feature Configuration

- Each feature which is not a required part of the agent's operation (i.e., the agent can function without it and the customer may have a reason to opt in or out of having the feature enabled) **MUST** have a corresponding configuration option to enable or disable it, in addition to any configuration options which allow adjusting various parameters relating to that feature.

- Every configuration option **MUST** exist as an API function in `newrelic/config_options.go` named `Config<FeatureName><ParameterName>(values)`. As a special case, functions to enable and disable features are named `Config<FeatureName>Enabled(bool)` and accept a single boolean parameter which will enable the feature if `true` or disable it if `false`. These option functions each return a value of type `newrelic.ConfigOption`, which is a function taking a pointer to a `newrelic.Config` structure which it will change in-place as appropriate to effect the option as requested. For example:

  ```go
  func ConfigRandomFeatureOfAwesomenessEnabled(enabled bool) ConfigOption {
      return func(cfg *Config) {
          cfg.AwesomeFeature.Enabled = enabled
      }
  }
  ```

- Every configuration option **SHOULD** exist as an environment variable that can be set by the user and parsed by the `newrelic/config_options.go` function `configFromEnvironment`. The environment variable **SHOULD** follow the same naming pattern as the API function but start with `NEW_RELIC_` and employ underscores between words (e.g., `NEW_RELIC_RANDOM_FEATURE_OF_AWESOMENESS_ENABLED` corresponds to `ConfigRandomFeatureOfAwesomenessEnabled`).
    - Use existing functions defined in `config_options.go` to parse environment values to ensure consistent handling of different data types such as booleans, lists, etc.

### Program Structure

- Functions **SHOULD** return early when conditions are found to indicate the function should abort, rather than creating long, deeply nested conditionals.
- The OAOO (Once And Only Once) principle **SHOULD** be applied wherever possible. Centralize common code which embodies data field handling, I/O, and other operations in preference to repeating similar code in multiple locations.
- Function adapters _[future section]_
- Export rules _[future section]_
- Parameter types and variables **SHOULD** be typed to the most appropriate interface to the job at hand. Don't ask for an `os.File` parameter if you just need an `io.ReadWriter` or `io.Writer`, for example.
- Conditional expressions **MUST** be expressed in a way that matches the way one would describe them in a natural language, to reduce confusion from awkward phrasing.
    - "Yoda conditionals" (testing the value of a variable in backwards order, e.g., `if 3 == x` instead of the preferred `if x == 3`) **MUST** be avoided.
        - Rationale: Originally designed to help avoid a common pitfall in C/C++ where a missing `=` (e.g., `if (x=3)` where `if (x==3)` was intended) introduces a problematic error; this practice serves no useful purpose in Go since the compiler disallows assignments in expressions outright. We are left with an awkward expression that impedes parsing of its true meaning, since the convention is to be asking about the value of the item on the left of the operator.
- Programs **MUST** exit via `os.Exit` or similar functions only in the `main` function, never elsewhere.
- Programs **MUST NOT** use `panic` as a way to choose to terminate on an error condition. Production code must never willingly panic.
- Function option parameters **SHOULD** be used when functions accept various parameters to set operational modes, rather than a complex "option" structure or long list of parameters. For example, instead of:

  ```go
  func Render(text string, bulletSet []rune, asHTML bool, asPostScript bool, compact bool) (string, error)
  ```

  define an option type with a set of option-setting functions:

  ```go
  text.Render(someString)
  text.Render(someString, AsHTML)
  text.Render(someString, AsPostScript)
  text.Render(someString, AsPostScript, WithBullets('*', 'o', '+'))
  text.Render(someString, AsPostScript, WithCompactText)
  ```

- In-line error tests **SHOULD** be used where they improve readability by making the code more concise, UNLESS the error or other values are needed outside the scope of the `if` statement:

  ```go
  // this
  if err := someFunction(); err != nil {
      ...
  }

  // as an alternative to this
  err := someFunction()
  if err != nil {
      ...
  }
  ```

- Map variables **SHOULD** be initialized as empty maps rather than nil maps. Prefer `myMap := make(map[string]int)` instead of `var myMap map[string]int`.
- Methods **MUST** use a pointer receiver if they could possibly modify the receiver.
- Global variables **MUST** be avoided wherever possible. Where they do appear, they **MUST** be immutable. Generally this is used for things like custom error values, never for global state.

### Error Handling

- All functions **MUST** signal error conditions if they run into an exception. Never silently ignore a situation which causes an unintended result. Never overload a non-error return value (e.g., returns the calculated integer value or `-1` in case of error).
- Functions **SHOULD** return errors compatible with their intended usage. Use specific types to support `errors.Is` or `errors.As` for callers who need to understand the actual error condition directly; use `fmt.Errorf` to create custom text string errors.
- Errors encountered by downstream functions **SHOULD** be wrapped properly when propagating them back to the caller. Pass the lower-level error value up without modification, or add your own context to it via `fmt.Errorf` and the `%w` and `%v` verbs.
- Error messages **SHOULD** be concise. Avoid redundant phrases such as `error:` or `system call failure:`. Simply state the reporting module (if appropriate) and what happened.
    - Avoid starting messages with capitals or ending them with punctuation.
- Error conditions **MUST** be resolved to the full extent possible, not simply reported, and **NEVER** completely ignored.

### Data Isolation

- Data **MUST** be copied at the boundaries between code fully under your control (e.g., within the API library you are writing) OR be treated as fully untrusted data.
    - Remember this applies to slices and maps inherently since they contain internal pointers to their underlying data.
    - Rationale: if anyone else holds a pointer to that data, they can change it without your knowledge or consent.
- Format strings to `fmt.Printf` and similar functions **MUST** be immutable and from a trusted local source, preferably a string literal value. Never print a string value from the user directly (as in `fmt.Printf(someStringValue)`).
    - Rationale: this has potentially severe security implications.
- When using templating packages such as `text/template` and various HTML/CSS-based frameworks, you **MUST** take care to ensure protection against XSS, CSRF, and other related vulnerabilities.
- When using SQL databases, care **MUST** be taken to fully guard against data from untrusted sources being included into SQL commands, leading to SQL injection vulnerabilities.
    - Corollary: This applies to all other similar contexts where data from one part of an application stack may end up interpreted as commands by another part.

### Concurrency

- Goroutines **MUST NOT** be started without providing a robust, clear way to terminate them and to detect when they have been terminated.
    - Ideally `context` **SHOULD** be used to provide this termination mechanism.
    - The termination mechanism **MUST** be independent from other data channels, so the order to terminate isn't blocked by other unread data. (Closing a data channel will work, however, since reading from a closed channel won't block.)
- Deferred functions (e.g. `Close()`) **MUST NOT** be deferred (as with `defer thing.Close()`) if that function might return an error value. Instead, defer a function literal which wraps a call to that function and checks properly for the error.
- Code **MUST** exercise caution with launching goroutines recursively. Goroutines do not have stack-size limitations and can overrun system resources if used carelessly.
- Loop index variables, and other variables whose values are calculated within the loop body itself, **MUST NOT** be placed directly into goroutines called within the loop. Instead, force a locally-scoped variable via `:=` within an iteration, or pass these volatile values as parameters to the function literal. The latter is the preferred method.
    - Rationale: values from the outer scope of a function literal are referenced inside the function literal with their current values. In a goroutine, since it is running concurrently those values may not be the ones you think they are.
- API libraries **SHOULD** avoid implementing concurrent operations behind the scenes. Prefer instead to let the application code call library routines from its own goroutines.
- `sync.Mutex` and `sync.RWMutex` values **SHOULD** never need to be pointers.
- A mutex used as part of a struct value **SHOULD** be a named member, not simply embedded by composition.

### Functionality

- Code which prepares data for use by other systems **SHOULD** write everything or nothing, avoiding partial writes of datasets. (E.g., make piecemeal writes to a buffer which is then flushed in one step when ready.)
- Compliance with API contracts **MUST** be verified at compile-time. For example, when implementing a local type `Frob` which must conform to the `frotz.Frob` interface, including a declaration such as `var _ frotz.Frob = (*Frob)(nil)` will ensure this compatibility.

## Preferred Notation

### Style

- Code **SHOULD** pass automated style checks by `golint` and `go vet`.

### Identifier Naming

- Names **SHOULD** be as short and simple as possible without sacrificing the clarity of the identifiers' meaning.
- General, short-lived error values **SHOULD** be captured into a variable named `err` whenever this does not introduce any other problems.
- Single letter variable names **MAY** be used for short loop iteration index variables, typically `i`, `j`, `k`, etc.
- Package names **SHOULD NOT** be repeated in the names of types and identifiers in that package (use name `gopher.Feed`, not `gopher.GopherFeed`).
- Identifiers **MUST** use camel case (medial capitals, as `descriptiveNameLikeThis`) and **MUST NOT** use snake case (`descriptive_name_like_this`).
    - The standard Go conventions do allow underscores in certain specific places such as test case grouping.
    - Variables which are global to a package **SHOULD** be named with a leading underscore (e.g., `_thing`) to make them easily identifiable. This does not apply to error names.
    - Identifiers which contain acronyms or initialisms such as `ID` or `URL` **MUST** keep a consistent case for that part of the name. For example, use `SetItemID` or `FetchURLFromServer` instead of `SetItemId` or `FetchUrlFromServer`.
- Error values at the global levels (such as exported custom errors from packages) **MUST** begin their names with `err` or `Err`.
- Custom error type names **MUST** end with `Error`.
- Package names **MUST** be all lower-case, simple, concise, informative, and singular.
- Identifier names **SHOULD NOT** include their type. Avoid, e.g., `sumInt`, `widgetMap`, `dataPayloadJSONString`, etc.

### Numeric Literals

- Numbers containing more than 3 digits **MAY** include underscores to separate the digits into groups as appropriate for the context (e.g., `123_456` instead of `123456`).
- Octal literals **MUST** be specified with a leading `0o`, not a plain leading `0`.

### Enums

- Enumerated values **SHOULD** begin with zero values where that corresponds naturally with the data type's zero value. Otherwise, they **SHOULD** begin with `1`.

### Types

- Type assertions **MUST** use the two-value form (`val, ok := thing.(string)`) and then check the value of `ok` before proceeding.
    - Rationale: the simpler form (`val := thing.(string)`) will panic if `thing` is not a string at that moment. We must avoid panics in production code.
- The data type `any` **MUST** be used instead of `interface{}`.

### Regular Expressions

- Compiling a regular expression **MUST** use the `MustCompile` method, so that an error in the expression itself is caught up-front as the application starts up instead of causing a panic at some random later time during the program's operation.
- Tip: Watch for regular expressions without explicit anchors (`^`, `$`, etc.); this may allow for any string containing the pattern to be matched, regardless of other text in the string (depending on the matching method being called).
- Regular expressions **SHOULD NOT** be used where a simpler, less-computationally-expensive function is already available (e.g., the `strconv` functions).

### Slices

- Returning an "empty" slice **MUST** be accomplished by returning a `nil` value.
- Checking a slice value for emptiness **MUST** be accomplished by checking its length (`len(theSlice) == 0`). Do not just check to see if it is `nil`.

### Time Values

- Values which note absolute or relative measurements of time **MUST** be based on the standard library `time` data types unless there is a compelling reason to use a custom type instead. Avoid storing dates or numbers of seconds as integers, for example.

### Documentation

- Programs **SHOULD** follow the standard godoc conventions to add fully-formed, coherent, readable, and informative documentation at the package, data type, and function level. If properly done, these are automatically pulled into public, customer-facing documentation.

## Testing

- New code added **MUST** be covered by corresponding unit tests which validate performance to the expected parameters in isolation from other units.
- Use of TDD techniques is **RECOMMENDED** to put initially-failing tests in place ahead of new feature code and bug fixes to ensure that adequate test coverage is provided and that the actual working code is inherently designed to be testable.
- Test cases **MUST** include security-related tests, including:
    - Edge cases, fuzzed inputs, and other inputs which may be completely unexpected by the code under test
    - Inputs which expose invalid assumptions about the validity or trustworthiness of data being read (e.g., unmarshalling data that could cause unintended consequences due to inappropriate trust in the data's marshalled format).
- Unit test cases **MUST** work offline without contact with database servers or other support services, and in isolation from other units.
- Integration and end-to-end product tests **MUST** test that units work as intended in concert with each other, and with other external services.
- Tests **MUST** be in separate files from production code. The test files **SHOULD** be named with the same base name as the corresponding source file, with a `_test.go` suffix.
- Tests **MUST NOT** panic wherever this can be avoided. It is preferable to use the unit testing framework to note a failing test as a failing test.

## New Language Features

- New language features (e.g., generics prior to Go 1.18) **MUST NOT** be used in production code until approval is given for the product to require the necessary version of Go. This avoids stranding customers who must run an older Go version. Once approved, the documentation and `go.mod` files **MUST** reflect the minimum Go version number to support the features used in the product.

## Handling Non-Conforming Legacy Code

- Code which does not conform to these guidelines **MUST** be corrected when touched as part of a bug fix, enhancement, or other code refactoring work. Otherwise, legacy code **MAY** be corrected as a separate task, as convenient, to reduce technical debt.

## Git Best Practices

### Branch Naming

- Branches **SHOULD** be named descriptively and convey the nature of the changes they contain as concisely as possible.

### Commit Messages

- When making commits, the developer **SHOULD** logically group changes by commit wherever possible. Avoid bulk commits of unrelated or non-atomic changes.
- When committing code that breaks patterns or otherwise requires explanation, a description of the change and the rationale behind the implementation **MUST** be provided in the commit message body.

### Updating Child Branches (rebase vs merge)

- When updating a child branch with upstream changes from a parent, a `rebase` action **MUST** be used over a merge. This preserves a linear history of changes without polluting a branch/PR with unrelated changes from other branches/PRs.

## Pull Requests

### Pull Request Naming

Pull request titles **SHOULD** follow Angular commit conventions:

```
<type>([optional scope]): subject
```

Example: `fix(internal): prioritize docker cgroup v2 detection`

Common `<type>` values:

- `feat` — adds a new feature or functionality
- `fix` — fixes a bug in existing functionality
- `test` — adds new tests or updates existing tests

The subject of the PR **SHOULD** be an imperative statement.

Examples:

- Bad: `fix: docker cgroup v2 detection`
- Good: `fix: prioritize docker cgroup v2 detection`
- Bad: `feat: nrpgx5`
- Good: `feat: add nrpgx5 support`

### Pull Request Description

A developer **MAY** include a description of the changes in the PR if the changes are considered non-trivial. If it is unclear whether a change is trivial, err on the side of caution and include a description. If the implementation breaks existing patterns or appears non-intuitive, include a statement on the "Why" something was done a particular way.

### Pull Request Content

A Pull Request **SHOULD** be atomic whenever possible. Unrelated changes **SHOULD** be avoided. Exceptions **MAY** be made in cases where a change to one system would cause failures or instability in another, seemingly unrelated system. Use best judgement and communicate directly with reviewers in this case.

### Pull Request Comments

- When a reviewer leaves a comment on a Pull Request, that comment **MUST** be resolved before the PR can be approved and merged.
- The resolution of a comment **MUST** be done either by the reviewer who left the comment, OR by GitHub if the comment was a code change suggestion that was accepted and applied by the PR owner. Comments **SHOULD NEVER** be resolved by the PR owner otherwise.
- When resolving a comment, a summary of the resolution (accepted/rejected and why) **MUST** be included as a final response to the comment IF the comment was discussed outside of the GitHub context (Slack, Zoom, etc.).

### Pull Request Tests

Before a PR can be merged into the `develop` or `master` branch, all CI/CD PR tests **MUST** be passing.

### Merging a Pull Request

A PR **MUST** have the following in order to be merged into `develop`:

- At least one approval from a Go Agent Team member
- All comments resolved
- All tests passing
- Latest upstream changes / no code conflicts (rebase if necessary)

If all the above are satisfied, the PR owner **MUST** use the **Squash and Merge** functionality to merge the PR. The options *Rebase and Merge* and *Create a merge commit* **MUST NEVER** be used — these options pollute the commit history of the `develop` branch.
