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
      | 200    | null         |
      | 200    | {}           |
      | 200    | {"tag_name":""} |

  Scenario Outline: Update archives match platform naming conventions
    When the update archive for "<os>" and "<arch>" at version "v1.0.0" is selected
    Then the archive filename should be "<filename>"

    Examples:
      | os      | arch  | filename                              |
      | windows | amd64 | AssetGoblin_v1.0.0_Windows_x64.zip    |
      | linux   | 386   | AssetGoblin_v1.0.0_Linux_x86.tar.gz   |
      | darwin  | arm64 | AssetGoblin_v1.0.0_macOS_arm64.tar.gz |
      | freebsd | amd64 | AssetGoblin_v1.0.0_Freebsd_x64.tar.gz |

  Scenario: Redirected download progress reports completion without cursor controls
    When a download of 100 bytes with advertised size 100 uses redirected progress and succeeds
    Then command output should contain "100% (100/100 bytes)"
    And command output should contain no terminal escape sequences
    And download progress should not refresh in place

  Scenario: Unknown archive sizes show bytes without a misleading percentage
    When a download of 100 bytes with advertised size -1 uses redirected progress and succeeds
    Then command output should contain "Downloading: 100 bytes"
    And command output should not contain "%"
    And command output should contain no terminal escape sequences

  Scenario: Failed downloads retain partial progress
    When a download of 25 bytes with advertised size 100 uses redirected progress and fails
    Then command output should contain "25% (25/100 bytes)"
    And command output should contain "Download failed."
    And command output should not contain "100%"

  Scenario: Terminal progress refreshes in place even with colors disabled
    When a download of 50 bytes with advertised size 100 uses terminal progress and succeeds
    Then command output should contain "50% (50/100 bytes)"
    And download progress should refresh in place
