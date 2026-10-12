Feature: Command-line operations
  Scenario Outline: Version and help flags work without starting the server
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should contain "<output>"

    Examples:
      | arguments                     | output              |
      | -version                      | development         |
      | -v                            | development         |
      | -help                         | plugins-dir         |
      |                               | Usage:              |
      | -serve=false                  | Usage:              |
      | -config                       | 8080                |
      | -clear-gob                    | Deleted gob config  |
      | -update                       | Already up to date. |
      | -plugins-dir=missing -version | development         |

  Scenario: Unknown flags fail with a usage error
    When AssetGoblin runs with arguments "-unknown-option"
    Then the command should exit with code 2
    And command output should contain "flag provided but not defined"

  Scenario: Plugin discovery defaults to the executable directory
    When the default plugin discovery directory is resolved
    Then it should be the executable directory

  Scenario: Explicit plugin discovery overrides the executable directory
    When plugin discovery directory "." is resolved
    Then it should be the working directory

  Scenario: Startup prints the branding and build version
    When AssetGoblin runs with arguments ""
    Then the startup banner should be present
    And the current build version should be reported

  Scenario Outline: Explicit false flags retain the default command behavior
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should contain "Usage:"

    Examples:
      | arguments        |
      | -version=false   |
      | -update=false    |
      | -config=false    |
      | -clear-gob=false |
