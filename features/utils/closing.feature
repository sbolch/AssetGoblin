Feature: Resource closing
  Scenario Outline: File and reader helpers release open handles
    Given an open temporary file
    When it is closed through the "<helper>" helper
    Then writing to the handle should fail

    Examples:
      | helper |
      | file   |
      | reader |

  Scenario: Closing an already closed reader remains safe
    Given an open temporary file
    When it is closed through the "reader" helper
    And it is closed through the "reader" helper
    Then writing to the handle should fail
