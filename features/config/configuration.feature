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
