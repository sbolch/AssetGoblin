Feature: Verified executable updates
  Scenario Outline: Published archive names install the stable release
    Given an updater release for "<os>" and "<arch>"
    When the release is installed into a temporary executable
    Then the update should succeed
    And the installed executable should contain "updated executable"
    And the download progress should finish successfully

    Examples:
      | os      | arch  |
      | linux   | amd64 |
      | linux   | 386   |
      | darwin  | arm64 |
      | windows | amd64 |

  Scenario Outline: Invalid updates leave the installed executable untouched
    Given an updater release for "linux" and "amd64"
    And the update fixture has "<problem>"
    When the release is installed into a temporary executable
    Then the update should fail
    And the installed executable should contain "original executable"

    Examples:
      | problem                   |
      | checksum mismatch         |
      | missing checksum entry    |
      | missing checksum document |
      | missing platform archive  |
      | no releases               |
      | damaged archive           |
      | download failure          |
      | cancelled download        |
      | interrupted download      |

  Scenario Outline: Updates respect semantic versions and development builds
    Given an updater release for "linux" and "amd64"
    And the current build is "<version>"
    Then an update should be <availability>

    Examples:
      | version     | availability |
      | development | available    |
      | v1.0.0      | available    |
      | v1.2.3      | unavailable  |
      | 1.2.3       | unavailable  |
      | v2.0.0      | unavailable  |

  Scenario: Interrupted archive reads report failed progress
    Given an updater release for "linux" and "amd64"
    And the update fixture has "interrupted download"
    When the release is installed into a temporary executable
    Then the update should fail
    And the installed executable should contain "original executable"
    And the download progress should report failure
