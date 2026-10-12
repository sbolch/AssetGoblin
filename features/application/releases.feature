Feature: Release metadata and update selection
  Scenario: Latest version comes from the release API response
    Given the release API responds with status 200 and body {"tag_name":"v1.0.0","body":"release notes","assets":[]}
    When the latest version is checked
    Then the latest version should be "v1.0.0"

  Scenario Outline: Failed or malformed API responses are reported
    Given the release API responds with status <status> and body <body>
    When the latest version is checked
    Then the version check should fail

    Examples:
      | status | body         |
      | 500    | server error |
      | 200    | invalid json |

  Scenario Outline: Update archives match platform naming conventions
    When the update archive for "<os>" and "<arch>" at version "v1.0.0" is selected
    Then the archive filename should be "<filename>"

    Examples:
      | os      | arch  | filename                              |
      | windows | amd64 | AssetGoblin_v1.0.0_Windows_x64.zip    |
      | linux   | 386   | AssetGoblin_v1.0.0_Linux_x86.tar.gz   |
      | darwin  | arm64 | AssetGoblin_v1.0.0_macOS_arm64.tar.gz |
      | freebsd | amd64 | AssetGoblin_v1.0.0_Freebsd_x64.tar.gz |

  Scenario: Checksums are selected by exact archive filename
    Given checksum lines "abcdef1234567890 archive.zip;1234567890abcdef other.tar.gz"
    When the checksum for "archive.zip" is selected
    Then its checksum should be "abcdef1234567890"

  Scenario: Missing archive checksums are reported
    Given checksum lines "1234567890abcdef other.tar.gz"
    When the checksum for "archive.zip" is selected
    Then its checksum should be missing

  Scenario: Malformed checksum lines are ignored
    Given checksum lines "invalid line format;abcdef1234567890 archive.zip"
    When the checksum for "archive.zip" is selected
    Then its checksum should be "abcdef1234567890"

  Scenario: The requested archive is selected from release assets
    Given a release asset "windows.zip" at "https://example.com/windows.zip"
    And a release asset "linux.tar.gz" at "https://example.com/linux.tar.gz"
    When the update URL for "windows.zip" is selected
    Then the selected asset URL should be "https://example.com/windows.zip"

  Scenario: A missing platform archive is reported
    Given a release asset "linux.tar.gz" at "https://example.com/linux.tar.gz"
    When the update URL for "windows.zip" is selected
    Then no matching asset should be found

  Scenario: The checksum document is selected from release assets
    Given a release asset "windows.zip" at "https://example.com/windows.zip"
    And a release asset "checksums.txt" at "https://example.com/checksums.txt"
    When the checksum URL is selected
    Then the selected asset URL should be "https://example.com/checksums.txt"

  Scenario: A missing checksum document is reported
    Given a release asset "windows.zip" at "https://example.com/windows.zip"
    When the checksum URL is selected
    Then no matching asset should be found
