Feature: Explicit optional runtime registration
  Scenario: Unregistered extensions are ignored
    Given an optional runtime plugin
    When plugin discovery runs
    Then 0 routes should be exposed

  Scenario: Registered backends use shared serving and lifecycle management
    Given an optional runtime plugin
    And its runtime is registered
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/optional/hello" with range ""
    Then the plugin response body should be "optional"
    When the manager is closed twice
    Then the optional runtime is closed once

  Scenario: Invalid backend manifests release the interpreter
    Given an optional runtime plugin
    And its runtime returns an invalid manifest
    And its runtime is registered
    When plugin discovery runs
    Then discovery should fail
    And the optional runtime is closed once

  Scenario: Python wins over a registered optional backend with the same name
    Given an optional runtime plugin
    And its runtime is registered
    And a Python plugin with the same name
    When plugin discovery runs
    Then 1 routes should be exposed
    And the optional runtime was not loaded

  Scenario: Duplicate backend registration fails
    Given its runtime is registered
    When its runtime is registered again
    Then backend registration should fail

  Scenario Outline: Invalid and reserved extensions cannot be registered
    Given a backend extension "<extension>"
    When its runtime is registered again
    Then backend registration should fail
    Examples:
      | extension |
      | .py |
      | runtime |
      | . |
      | ../runtime |
      | .a.b |
