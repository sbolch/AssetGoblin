Feature: Python plugin discovery and validation
  Scenario: Unrelated files do not become plugins
    Given an unrelated Python file
    When plugin discovery runs
    Then 0 routes should be exposed

  Scenario Outline: Invalid plugin manifests reject startup
    Given plugin source <source>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | source                                                                  |
      | plugin = {                                                                |
      | value = 1                                                             |
      | plugin = {'api_version': 2, 'routes': []}                                        |
      | plugin = {'api_version': 1, 'routes': []}                                        |
      | plugin = {'api_version': 1, 'routes': [{'path': '/bad', 'directory': '.'}]}             |
      | plugin = {'api_version': 1, 'routes': [{'path': '/{bad}/', 'directory': '.'}]}          |
      | plugin = {'api_version': 1, 'routes': [{'path': '/a/'}]}                            |
      | plugin = {'api_version': 1, 'routes': [{'path': '/a/', 'handle': True}]}                |
      | raise RuntimeError('initialization failure')                                            |
      | open('nonexistent-assetgoblin-test-file')                                                      |
      | plugin = {'api_version': 1, 'extends': 'missing', 'extend': lambda p: p} |
      | plugin = {'api_version': 1, 'extends': 'custom', 'extend': lambda p: p}  |

  Scenario: A missing discovery directory reports an error
    When discovery runs in a missing directory
    Then discovery should fail

  Scenario: Discovery ordering is stable and the route registry is immutable
    Given plugins named "b,a" with matching routes
    When plugin discovery runs
    Then routes should be ordered "/a/,/b/"
    When a returned route list is changed
    Then routes should be ordered "/a/,/b/"

  Scenario: Duplicate plugin routes reject startup
    Given plugins named "a,b" with matching routes
    And plugin "duplicate" declares route "/a/"
    When plugin discovery runs
    Then discovery should fail

  Scenario Outline: Initialization cannot exceed script resource limits
    Given an initialization script exceeding "<limit>"
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | limit |
      | time  |
      | size  |

  Scenario Outline: Image codecs require valid names and both callbacks
    Given image codec definition <definition>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | definition                                                                   |
      | 'foo': {'mime_type': 'image/foo', 'encode': lambda r: {}}                             |
      | '../foo': {'mime_type': 'image/foo', 'encode': lambda r: {}, 'decode': lambda r: {}} |
      | 'foo': {'mime_type': 'text/foo', 'encode': lambda r: {}, 'decode': lambda r: {}}         |
