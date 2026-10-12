Feature: Image discovery and transformation options
  Scenario Outline: Direct dimensions are parsed and validated
    When size "<input>" is selected with fit "<fit>"
    Then the resize geometry should be "<geometry>" with sizing "<sized>"
    And the selected fit should be "<selected>"
    And the selection should be a preset "false"

    Examples:
      | input    | fit   | geometry | sized | selected |
      | 640      |       | 640      | true  | contain  |
      | 640x480  |       | 640x480  | true  | contain  |
      | 640x480  | cover | 640x480  | true  | cover    |
      | 640x480x |       |          | false |          |
      | abc      |       |          | false |          |
      | 640xabc  |       |          | false |          |
      | 0        |       |          | false |          |
      | -100     |       |          | false |          |
      | 640x0    |       |          | false |          |
      |          |       |          | false |          |

  Scenario Outline: Named presets preserve all transformation settings
    Given preset "custom" has settings <settings>
    When size "custom" is selected with fit "<fit>"
    Then the resize geometry should be "<geometry>" with sizing "true"
    And the selection should be a preset "true"
    And the transformation settings should match <settings>

    Examples:
      | settings                                     | fit   | geometry |
      | {"width":200,"height":200,"fit":"cover"}     |       | 200x200  |
      | {"width":960,"fit":"contain"}                |       | 960      |
      | {"width":640,"rotate":90}                    |       | 640      |
      | {"width":640,"flip":"horizontal"}            |       | 640      |
      | {"width":640,"brightness":50}                |       | 640      |
      | {"width":640,"contrast":-25}                 |       | 640      |
      | {"width":640,"gamma":1.5}                    |       | 640      |
      | {"width":640,"filters":["grayscale","blur"]} |       | 640      |
      | {"width":200,"height":200,"fit":"cover"}     | cover | 200x200  |

  Scenario: Direct sizes still work when presets are configured
    Given preset "custom" has settings {"width":200,"height":200}
    When size "800" is selected with fit ""
    Then the resize geometry should be "800" with sizing "true"
    And the selection should be a preset "false"

  Scenario Outline: Sources are discovered in configured format order
    Given supported source formats "<formats>"
    And a source file "test.jpg"
    And a source file "test.png"
    When the source basename "<basename>" is searched
    Then the discovered filename should be "<filename>"

    Examples:
      | formats | basename | filename |
      | jpg,png | test     | test.jpg |
      | png     | test     | test.png |
      | jpg,png | missing  |          |
      |         | test     |          |

  Scenario: A directory is never treated as an image source
    Given supported source formats "jpg"
    And a source directory "source.jpg"
    When the source basename "source" is searched
    Then the discovered filename should be ""

  Scenario Outline: Output formats accept an optional leading dot
    Given supported source formats "<formats>"
    When output format "<format>" is checked
    Then the format should be supported "<supported>"

    Examples:
      | formats      | format | supported |
      | jpg,png,webp | .jpg   | true      |
      | jpg,png,webp | jpg    | true      |
      | jpg,png,webp | .gif   | false     |
      | jpg,png,webp |        | false     |
      |              | .jpg   | false     |

  Scenario Outline: Relative paths resolve under the working directory
    When image path "<path>" is resolved
    Then it should resolve to "<resolved>" under the working directory

    Examples:
      | path            | resolved      |
      | relative/path   | relative/path |
      | ./relative/path | relative/path |

  Scenario: Absolute paths retain their location
    When an absolute image path is resolved
    Then it should retain its absolute location
