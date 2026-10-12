Feature: Plugin composition and cache revisions
  Scenario: A leaf composes its complete inheritance chain regardless of filename order
    Given a three-plugin inheritance chain
    When plugin discovery runs
    Then 1 routes should be exposed
    When a plugin "GET" request is sent to "/base/file" with range ""
    Then the plugin response body should be "base:middle:child"

  Scenario: Editing a parent invalidates the leaf's cache revision
    Given a three-plugin inheritance chain
    When plugin discovery runs
    And its leaf revision is remembered
    And the parent script is changed
    And plugin discovery runs
    Then its leaf revision should differ

  Scenario: Forked extensions cannot register conflicting inherited routes
    Given a three-plugin inheritance chain
    And another extension of the middle plugin
    When plugin discovery runs
    Then discovery should fail

  Scenario Outline: Extension callbacks must return valid manifests
    Given a base plugin
    And extension source <source>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | source                                                                   |
      | return {api_version=1,extends='base'}                                    |
      | return {api_version=1,extends='base',extend=function(p)return false end} |
