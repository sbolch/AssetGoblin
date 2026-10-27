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
      | -h                            | plugins-dir         |
      |                               | Usage:              |
      | -serve=false                  | Usage:              |
      | -config                       | 8080                |
      | -clear-gob                    | Deleted gob config  |
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
      | -S=false         |
      | -S -serve=false  |
      | -serve -S=false  |

  Scenario Outline: Both serve flags start plugin discovery
    When AssetGoblin runs with arguments "<arguments> -plugins-dir=missing"
    Then the command should exit with code 1
    And command output should contain "Failed to load plugins"

    Examples:
      | arguments |
      | -serve    |
      | -S        |

  Scenario: Configuration tables retain Unicode values and sorted presets without colors
    Given CLI colors are disabled
    And CLI configuration contains {"port":"9090","public_dir":"静的 assets","image":{"presets":{"wide":{"width":640,"height":480,"gamma":1},"small":{"width":80,"gamma":1}}}}
    When AssetGoblin runs with arguments "-clear-gob"
    And AssetGoblin runs with arguments "-config"
    Then the command should exit with code 0
    And command output should contain "Setting"
    And command output should contain "Value"
    And command output should contain "9090"
    And command output should contain "静的 assets"
    And command output should contain "small=80, wide=640x480"
    And command output should contain no terminal escape sequences

  Scenario Outline: Both CLI entry points format complete update notifications without colors
    Given CLI colors are disabled
    And the release API responds with status 200 and body {"tag_name":"v123.456.789-preview-with-a-long-version-name","assets":[]}
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should contain "Update available: v123.456.789-preview-with-a-long-version-name"
    And command output should contain no terminal escape sequences

    Examples:
      | arguments |
      | -version  |
      |           |

  Scenario Outline: Unavailable release checks do not display an empty update notification
    Given the release API responds with status 503 and body unavailable
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should not contain "Update available:"

    Examples:
      | arguments |
      | -version  |
      |           |

  Scenario Outline: Help entry points share the flag table without colors
    Given CLI colors are disabled
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should contain "Usage:"
    And command output should contain "Flag"
    And command output should contain "Description"
    And command output should contain "Default"
    And command output should contain "-h, -help"
    And command output should contain "-S"
    And command output should contain "-serve"
    And command output should contain "-plugins-dir string"
    And command output should not contain "-python string"
    And command output should contain no terminal escape sequences

    Examples:
      | arguments |
      | -h        |
      | -help     |
      |           |

  Scenario Outline: Explicit help uses the standard diagnostic stream
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command stdout should be empty
    And command output should contain "Usage:"

    Examples:
      | arguments |
      | -h        |
      | -help     |

  Scenario Outline: Version output retains its plain first line when redirected
    Given CLI colors are disabled
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And the first output line should be the current version
    And command output should contain "Build:"
    And command stderr should be empty
    And command output should contain no terminal escape sequences

    Examples:
      | arguments |
      | -version  |
      | -v        |

  Scenario Outline: Success messages remain readable without colors
    Given the CLI build version is "v1.0.0"
    And CLI colors are disabled
    When AssetGoblin runs with arguments "<arguments>"
    Then the command should exit with code 0
    And command output should contain "<message>"
    And command stderr should be empty
    And command output should contain no terminal escape sequences

    Examples:
      | arguments  | message                                |
      | -clear-gob | Deleted gob config cache (if it existed) |
      | -update    | Already up to date.                    |

  Scenario: Startup summaries show configuration and sorted plugin capabilities
    Given a startup summary for configuration {"port":"8080","PublicDir":"public","secret":"private-secret","image":{"formats":["png"],"path":"/img/","directory":"assets/img","CacheDir":"cache/img"}}
    Then command output should contain "Starting AssetGoblin"
    And command output should contain ":8080"
    And command output should contain "public"
    And command output should contain "assets/img"
    And command output should contain "cache/img"
    And command output should contain "/audio/, /video/"
    And command output should contain "qoi, zoi"
    And command output should not contain "private-secret"
    And command output should contain no terminal escape sequences

  Scenario: Startup summaries identify disabled image processing
    Given a startup summary for configuration {"port":"8080","PublicDir":"public","image":{"directory":"unused-image-directory"}}
    Then command output should contain "Disabled"
    And command output should not contain "unused-image-directory"

  Scenario: Release notes wrap long lines and retain the original words
    When release notes "Fixed plugin discovery and improved image conversions." are rendered at width 20
    Then command output should contain "Release notes:"
    And command output should contain "discovery"
    And command output should contain "conversions."
    And every output line should fit within 20 columns
    And command output should contain no terminal escape sequences

  Scenario: Empty release notes produce no heading
    When release notes "" are rendered at width 80
    Then command output should not contain "Release notes:"

  Scenario: The interpreter runtime flag is no longer accepted
    When AssetGoblin runs with arguments "-python=missing -version"
    Then the command should exit with code 2
    And command output should contain "flag provided but not defined"

  Scenario: The effective configuration shows the Python interpreter
    Given CLI configuration contains {"python_executable":"/opt/plugin-env/bin/python"}
    When AssetGoblin runs with arguments "-clear-gob"
    And AssetGoblin runs with arguments "-config"
    Then the command should exit with code 0
    And command output should contain "python_executable"
    And command output should contain "/opt/plugin-env/bin/python"

  Scenario: Serving uses the configured Python interpreter
    Given CLI configuration contains {"python_executable":"./missing-configured-python"}
    And a CLI Python plugin
    When AssetGoblin runs with arguments "-clear-gob"
    And AssetGoblin runs with arguments "-serve -plugins-dir=."
    Then the command should exit with code 1
    And command output should contain "start Python"
    And command output should contain "missing-configured-python"

  Scenario: Update failures return a failing exit status
    Given the release API responds with status 503 and body unavailable
    When AssetGoblin runs with arguments "-update"
    Then the command should exit with code 1
    And command output should contain "Failed to update"
    And command output should not contain "Update finished successfully."

  Scenario: Missing releases do not report a successful update
    Given the release API responds with status 200 and body []
    When AssetGoblin runs with arguments "-update"
    Then the command should exit with code 1
    And command output should contain "no compatible release available"
