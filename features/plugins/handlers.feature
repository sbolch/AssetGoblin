Feature: Root-scoped plugin files and Lua callbacks
  Scenario Outline: File plugins honor HTTP semantics and reject unsafe paths
    Given a rooted file plugin with sample content "abcdef"
    When a plugin "<method>" request is sent to "<path>" with range "<range>"
    Then the plugin response status should be <status>

    Examples:
      | method | path             | range     | status |
      | GET    | /a/sample.bin    |           | 200    |
      | HEAD   | /a/sample.bin    |           | 200    |
      | GET    | /a/sample.bin    | bytes=1-3 | 206    |
      | POST   | /a/sample.bin    |           | 405    |
      | GET    | /a/missing       |           | 404    |
      | GET    | /a/nested        |           | 404    |
      | GET    | /a/../secret.bin |           | 404    |

  Scenario: File plugin byte ranges retain their requested content
    Given a rooted file plugin with sample content "abcdef"
    When a plugin "GET" request is sent to "/a/sample.bin" with range "bytes=1-3"
    Then the plugin response status should be 206
    And the plugin response body should be "bcd"

  Scenario: Symlinks cannot escape the plugin file root
    Given a rooted file plugin with sample content "abcdef"
    And a source symlink pointing outside its root
    When a plugin "GET" request is sent to "/a/escape.bin" with range ""
    Then the plugin response status should be 404

  Scenario: Original file responses support conditional requests
    Given a rooted file plugin with sample content "abcdef"
    When a plugin "GET" request is sent to "/a/sample.bin" with range ""
    And the same request is made with its last-modified condition
    Then the plugin response status should be 304

  Scenario: A Lua callback receives method, path, query and headers
    Given a custom callback with a persistent counter
    And request header "X-Test" is "value"
    When a plugin "GET" request is sent to "/custom/hello?name=world" with range ""
    Then the plugin response status should be 201
    And the plugin response body should be "GET:hello:world:value"
    And response header "X-Count" should be "1"
    When a plugin "HEAD" request is sent to "/custom/hello?name=world" with range ""
    Then the plugin response status should be 201
    And the callback response should have an empty body and positive content length
    And response header "X-Count" should be "2"

  Scenario Outline: Callback failures are handled without exposing internal errors
    Given a custom callback with a persistent counter
    When a plugin "GET" request is sent to "/custom/<path>" with range "bytes=1-3"
    Then the plugin response status should be <status>

    Examples:
      | path    | status |
      | file    | 206    |
      | escape  | 404    |
      | error   | 500    |
      | invalid | 500    |
      | header  | 500    |

  Scenario: A callback can map a URL to a root-scoped file
    Given a custom callback with a persistent counter
    When a plugin "GET" request is sent to "/custom/file" with range "bytes=1-3"
    Then the plugin response body should be "bcd"

  Scenario: Cancellation releases the interpreter for subsequent requests
    Given a custom callback with a persistent counter
    When a looping callback request is cancelled
    Then the plugin response status should be 500
    When a plugin "GET" request is sent to "/custom/hello" with range ""
    Then the plugin response status should be 201

  Scenario: Closing the plugin manager is idempotent and disables callbacks
    Given a custom callback with a persistent counter
    When the manager is closed twice
    And a plugin "GET" request is sent to "/custom/hello" with range ""
    Then the plugin response status should be 500

  Scenario: Concurrent requests serialize counter updates without losing responses
    Given a counting callback
    When 20 plugin requests arrive concurrently
    Then every response should contain a different counter from 1 to 20
