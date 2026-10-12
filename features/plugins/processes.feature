Feature: Generic external tools and process pipelines
  Scenario: Scripts define custom process logic without media knowledge in core
    Given an uppercase tool plugin
    When a plugin "GET" request is sent to "/custom/source.txt" with range ""
    Then the plugin response status should be 200
    And the plugin response body should be "CUSTOM PLUGIN LOGIC"

  Scenario: Scripts cannot invoke undeclared tools
    Given an uppercase tool plugin without a tool declaration
    When a plugin "GET" request is sent to "/custom/source.txt" with range ""
    Then the plugin response status should be 500

  Scenario: Identical process requests reuse their cached output
    Given process input "custom plugin logic"
    When the "upper" tool runs with revision "revision1"
    Then the process output should be "CUSTOM PLUGIN LOGIC"
    When the output path is remembered
    And the "upper" tool runs with revision "revision1"
    Then the output path should be unchanged

  Scenario: Changing a script revision invalidates cached output
    Given process input "custom plugin logic"
    When the "upper" tool runs with revision "revision1"
    And the output path is remembered
    And the "upper" tool runs with revision "revision2"
    Then the output path should differ
    And 2 completed outputs and no temporary files should remain

  Scenario Outline: Failed and cancelled processes publish no partial outputs
    Given process input "custom plugin logic"
    When the "<mode>" tool runs with revision "revision1"
    Then processing should fail
    And 0 completed outputs and no temporary files should remain

    Examples:
      | mode |
      | fail |
      | wait |

  Scenario Outline: Process plans reject invalid executables, extensions and arguments
    Given process plan <plan>
    When the process plan is validated
    Then processing should fail

    Examples:
      | plan                                                                           |
      | {"tool":"../tool","extension":"txt","args":["{input}","{output}"]}             |
      | {"tool":"tool","extension":"../txt","args":["{input}","{output}"]}             |
      | {"tool":"tool","extension":"txt","args":["missing-placeholders"]}              |
      | {"tool":"tool","extension":"txt","args":["{input}","{output}","bad\u0000arg"]} |

  Scenario: Pipeline stages consume the previous output
    Given process input "custom stages"
    When two uppercase stages run as a declared pipeline
    Then the process output should be "CUSTOM STAGES"

  Scenario: Pipeline stages cannot invoke undeclared tools
    Given process input "custom stages"
    When two uppercase stages run without a tool declaration
    Then processing should fail
