Feature: Lua plugin discovery and validation
  Scenario: Unrelated files do not become plugins
    Given an unrelated Lua file
    When plugin discovery runs
    Then 0 routes should be exposed

  Scenario Outline: Invalid plugin manifests reject startup
    Given plugin source <source>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | source                                                                  |
      | return {                                                                |
      | local x = 1                                                             |
      | return {api_version=2,routes={}}                                        |
      | return {api_version=1,routes={}}                                        |
      | return {api_version=1,routes={{path='/bad',directory='.'}}}             |
      | return {api_version=1,routes={{path='/{bad}/',directory='.'}}}          |
      | return {api_version=1,routes={{path='/a/'}}}                            |
      | return {api_version=1,routes={{path='/a/',handle=true}}}                |
      | os.execute('echo forbidden')                                            |
      | dofile('anything')                                                      |
      | return {api_version=1,extends='missing',extend=function(p)return p end} |
      | return {api_version=1,extends='custom',extend=function(p)return p end}  |

  Scenario: A missing discovery directory reports an error
    When discovery runs in a missing directory
    Then discovery should fail

  Scenario: Discovery ordering is stable and the route registry is immutable
    Given plugins named "b,a" with matching routes
    When plugin discovery runs
    Then routes should be ordered "/a/,/b/"
    When a returned route list is changed
    Then routes should be ordered "/a/,/b/"

  Scenario: Duplicate plugin routes reject startup
    Given plugins named "a,b" with matching routes
    And plugin "duplicate" declares route "/a/"
    When plugin discovery runs
    Then discovery should fail

  Scenario Outline: Initialization cannot exceed script resource limits
    Given an initialization script exceeding "<limit>"
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | limit |
      | time  |
      | size  |

  Scenario Outline: Image codecs require valid names and both callbacks
    Given image codec definition <definition>
    When plugin discovery runs
    Then discovery should fail

    Examples:
      | definition                                                                   |
      | foo={mime_type='image/foo',encode=function()end}                             |
      | ['../foo']={mime_type='image/foo',encode=function()end,decode=function()end} |
      | foo={mime_type='text/foo',encode=function()end,decode=function()end}         |
