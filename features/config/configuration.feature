Feature: Configuration and persistent cache
  As an operator
  I want predictable defaults and cached configuration
  So that startup and configuration changes behave consistently

  Scenario: Missing configuration uses the application defaults
    When configuration is loaded
    Then the default configuration should be used
    And configuration should be marked as loaded from disk

  Scenario: Disk configuration overrides defaults and records its source
    Given the configuration file contains {"port":"9091","public_dir":"from-file"}
    When configuration is loaded
    Then the configured port should be "9091"
    And the public directory should be "from-file"
    And the configuration source should be the configuration file
    And configuration should be marked as loaded from disk

  Scenario: Cached configuration survives a complete serialization round trip
    Given a complete custom configuration
    When configuration is saved to the gob cache
    Then the gob cache file should exist
    When configuration is read directly from the gob cache
    Then every configuration field should match the saved configuration

  Scenario: Startup prefers cached configuration to a changed file
    Given a complete custom configuration
    When configuration is saved to the gob cache
    Given the configuration file contains {"port":"9999"}
    When configuration is loaded
    Then the configured port should be "9090"
    And every configuration field should match the saved configuration
    And configuration should be marked as loaded from gob

  Scenario: Clearing the cache allows changed disk configuration to take effect
    Given the configuration file contains {"port":"9091"}
    When configuration is loaded
    Given the configuration file contains {"port":"9092"}
    When the gob cache is removed
    And configuration is loaded
    Then the configured port should be "9092"
    And configuration should be marked as loaded from disk

  Scenario: A missing gob cache reports a read error
    When configuration is read directly from the gob cache
    Then configuration loading should fail

  Scenario: Cache removal is safe when repeated
    Given a complete custom configuration
    When configuration is saved to the gob cache
    And the gob cache is removed
    Then the gob cache file should not exist
    When the gob cache is removed
    Then the gob cache file should not exist

  Scenario: Configuration search starts in the working directory
    When configuration lookup paths are resolved
    Then the working directory should be searched first
    And the user configuration directory should be searched

  Scenario: Cache location uses the OS user cache directory
    When the default cache directory is resolved
    Then it should be the application directory under the OS user cache

  Scenario Outline: Python interpreter configuration survives caching
    Given the configuration file contains {"python_executable":"<executable>"}
    When configuration is loaded
    Then the Python executable should be "<executable>"
    When configuration is read directly from the gob cache
    Then the Python executable should be "<executable>"

    Examples:
      | executable |
      | |
      | /opt/plugin environment/bin/python |

  Scenario: Reloading replaces fields removed from disk configuration
    Given the configuration file contains {"python_executable":"/old/python", "public_dir":"old-public"}
    When configuration is loaded
    And the gob cache is removed
    Given the configuration file contains {"port":"9092"}
    When configuration is loaded
    Then the configured port should be "9092"
    And the Python executable should be ""
    And the public directory should be "public"

  Scenario: Presets default to contain sizing and neutral gamma
    Given the configuration file contains {"image":{"presets":{"small":{"width":100}}}}
    When configuration is loaded
    Then configuration should be marked as loaded from disk
    And preset "small" should have settings {"width":100,"fit":"contain","gamma":1}
    When configuration is read directly from the gob cache
    Then preset "small" should have settings {"width":100,"fit":"contain","gamma":1}

  Scenario Outline: Invalid preset values fail configuration loading
    Given the configuration file contains {"image":{"presets":{"invalid":<preset>}}}
    When configuration is loaded
    Then configuration loading should fail

    Examples:
      | preset |
      | {"width":0} |
      | {"width":100,"rotate":45} |
      | {"width":100,"flip":"diagonal"} |
      | {"width":100,"crop":"unknown"} |
      | {"width":100,"brightness":101} |
      | {"width":100,"contrast":-101} |
      | {"width":100,"gamma":0.05} |
      | {"width":100,"gamma":11} |
      | {"width":100,"filters":["unknown"]} |
