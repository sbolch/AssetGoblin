Feature: Python plugin runtime
  Scenario: Unrelated Python scripts are ignored
    Given an unrelated Python file
    When plugin discovery runs
    Then 0 routes should be exposed

  Scenario Outline: Invalid Python manifests reject startup
    Given Python plugin source <source>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | source |
      | plugin = { |
      | value = 1 |
      | plugin = {'api_version': True, 'routes': []} |
      | plugin = {'api_version': 2, 'routes': []} |
      | plugin = {'api_version': 1, 'routes': []} |
      | plugin = {'api_version': 1, 'routes': 'bad'} |
      | plugin = {'api_version': 1, 'tools': {}, 'routes': [{'path': '/a/', 'directory': '.'}]} |
      | plugin = {'api_version': 1, 'routes': [{'path': '/a/', 'handle': {'$callback': '0'}}]} |
      | plugin = {'api_version': 1, 'routes': [{'path': '/bad', 'directory': '.'}]} |
      | plugin = {'api_version': 1, 'routes': [{'path': '/a/', 'handle': True}]} |
      | plugin = {'api_version': 1, 'routes': [{'path': '/a/'}]} |
      | plugin = {'api_version': 1, 'tools': ['../tool'], 'routes': [{'path': '/a/', 'directory': '.'}]} |
      | plugin = {'api_version': 1, 'extends': 'missing', 'extend': lambda p: p} |
      | plugin = {'api_version': 1, 'extends': 'custom', 'extend': lambda p: p} |
      | plugin = {'api_version': 1, 'image_codecs': {'custom': {'mime_type': 'text/plain', 'decode': lambda r: {}, 'encode': lambda r: {}}}} |

  Scenario: Unregistered runtime files with the same name are ignored
    Given Python plugin source plugin = {'api_version': 1, 'routes': [{'path': '/python/', 'handle': lambda r: {'body': 'Python'}}]}
    And an unregistered plugin with the same name
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/python/test" with range ""
    Then the plugin response body should be "Python"

  Scenario: Independently named unregistered plugins are ignored
    Given Python plugin source plugin = {'api_version': 1, 'routes': [{'path': '/python/', 'directory': '.'}]}
    And a separate unregistered plugin
    When plugin discovery runs
    Then 1 routes should be exposed

  Scenario: Unregistered runtime files do not start Python
    Given an unregistered plugin
    When discovery uses an unavailable Python interpreter
    Then 0 routes should be exposed

  Scenario: A missing configured Python interpreter rejects Python plugins
    Given Python plugin source plugin = {'api_version': 1, 'routes': [{'path': '/python/', 'directory': '.'}]}
    When discovery uses an unavailable Python interpreter
    Then discovery should fail

  Scenario: Initialization has a deadline
    Given an unbounded Python initialization
    When plugin discovery runs
    Then discovery should fail

  Scenario: Source size is bounded
    Given an oversized Python source
    When plugin discovery runs
    Then discovery should fail

  Scenario: Python receives request metadata and keeps state without stdout corruption
    Given a Python callback with a persistent counter
    And request header "X-Test" is "value"
    When a plugin "GET" request is sent to "/custom/hello?name=world" with range ""
    Then the plugin response status should be 201
    And the plugin response body should be "GET:hello:world:value"
    And response header "X-Count" should be "1"
    When a plugin "HEAD" request is sent to "/custom/hello?name=world" with range ""
    Then the callback response should have an empty body and positive content length
    And response header "X-Count" should be "2"

  Scenario Outline: Responses obey host validation and root-scoped file serving
    Given a Python callback with a persistent counter
    When a plugin "GET" request is sent to "/custom/<path>" with range "bytes=1-3"
    Then the plugin response status should be <status>

    Examples:
      | path | status |
      | file | 206 |
      | escape | 404 |
      | error | 500 |
      | crash | 500 |
      | invalid | 500 |
      | header | 500 |
      | oversized | 500 |
      | protocol | 500 |

  Scenario: Callback exceptions leave the worker available
    Given a Python callback with a persistent counter
    When a plugin "GET" request is sent to "/custom/error" with range ""
    Then the plugin response status should be 500
    When a plugin "GET" request is sent to "/custom/hello" with range ""
    Then the plugin response status should be 201

  Scenario: Cancellation terminates a blocked worker and subsequent callbacks fail promptly
    Given a Python callback with a persistent counter
    When a looping callback request is cancelled
    Then the plugin response status should be 500
    When a plugin "GET" request is sent to "/custom/hello" with range ""
    Then the plugin response status should be 500

  Scenario: Closing the manager twice stops workers and disables callbacks
    Given a Python callback with a persistent counter
    When the manager is closed twice
    And a plugin "GET" request is sent to "/custom/hello" with range ""
    Then the plugin response status should be 500

  Scenario: Concurrent callbacks serialize persistent state
    Given a Python counting callback
    When 20 plugin requests arrive concurrently
    Then every response should contain a different counter from 1 to 20

  Scenario: Python extensions compose parent callbacks and register only leaf routes
    Given a Python inheritance chain
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/base/file" with range ""
    Then the plugin response body should be "base:middle:child"

  Scenario: Parent edits invalidate Python leaf revisions
    Given a Python inheritance chain
    When plugin discovery runs
    And its leaf revision is remembered
    And the Python parent is changed
    And plugin discovery runs
    Then its leaf revision should differ

  Scenario: Python callbacks reuse the generic transformation capability
    Given a Python uppercase tool plugin
    When a plugin "GET" request is sent to "/custom/source.txt" with range ""
    Then the plugin response status should be 200
    And the plugin response body should be "CUSTOM PLUGIN LOGIC"

  Scenario: Python manifests register image codecs
    Given a valid Python codec
    When plugin discovery runs
    Then the Python codec is registered

  Scenario: Worker pipes preserve Unicode requests and responses
    Given a Python callback with a persistent counter
    When a plugin "GET" request is sent to "/custom/hello?name=%E9%9D%99%E7%9A%84" with range ""
    Then the plugin response body should be "GET:hello:静的:"

  Scenario Outline: Python callback results obey shared response validation
    Given Python plugin source plugin = {'api_version': 1, 'routes': [{'path': '/test/', 'handle': lambda r: <response>}]}
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/test/hello" with range ""
    Then the plugin response status should be 500

    Examples:
      | response |
      | [] |
      | {'status': True} |
      | {'status': 201.5} |
      | {'status': 204, 'body': 'bad'} |
      | {'headers': {'Content-Length': '10'}} |
      | {'headers': {'X-Test': 10}} |
      | {'file': 'test', 'body': 'bad'} |
      | {'process': {'tool': 'tool', 'extension': 'txt', 'args': ['{input}', '{output}']}} |
      | {'file': 'test', 'process': {'tool': 'tool', 'extension': 'txt', 'args': ['{input}', 1]}} |
      | {'file': 'test', 'processes': []} |
      | {'transform': {}} |
      | {'file': 'test', 'process': {'tool': 'tool', 'extension': 'txt', 'args': ['{input}', '{output}']}, 'processes': [{'tool': 'tool', 'extension': 'txt', 'args': ['{input}', '{output}']}]} |

  Scenario: Conflicting Python routes fail startup
    Given Python plugin source plugin = {'api_version': 1, 'routes': [{'path': '/test/', 'directory': '.'}, {'path': '/test/', 'directory': '.'}]}
    When plugin discovery runs
    Then discovery should fail

  Scenario: Python extensions cannot inherit from unregistered plugins
    Given an unregistered plugin
    And Python plugin source plugin = {'api_version': 1, 'extends': 'external', 'extend': lambda p: p}
    When plugin discovery runs
    Then discovery should fail

  Scenario: Python plugins support normal module introspection
    Given a Python plugin using a dataclass
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/module/test" with range ""
    Then the plugin response body should be "module works"

  Scenario: A selected virtual environment supplies installed plugin dependencies
    Given a Python plugin using an installed virtual environment package
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/venv/test" with range ""
    Then the plugin response body should be "installed package"
