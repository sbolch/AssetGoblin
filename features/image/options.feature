Feature: Image discovery and transformation options
  Scenario Outline: Direct dimensions are parsed and validated
    When size "<input>" is selected with fit "<fit>"
    Then the resize geometry should be "<geometry>" with sizing "<sized>"
    And the selected fit should be "<selected>"

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

  Scenario Outline: Transformations produce the requested image dimensions and pixels
    Given a four-color image source
    When image URL "<url>" is requested
    Then the image should be <width> by <height> with first pixel "<pixel>"

    Examples:
      | url                                                 | width | height | pixel       |
      | /img/50/colors.png                                  | 50    | 40     | 255,0,0     |
      | /img/50x20/colors.png                               | 25    | 20     | 255,0,0     |
      | /img/50x20/colors.png?fit=cover                     | 50    | 20     | 255,0,0     |
      | /img/50/colors.png?rotate=90                        | 40    | 50     | 0,0,255     |
      | /img/50/colors.png?flip=horizontal                  | 50    | 40     | 0,255,0     |
      | /img/50/colors.png?flip=vertical                    | 50    | 40     | 0,0,255     |
      | /img/50/colors.png?flip=both                        | 50    | 40     | 255,255,0   |
      | /img/50/colors.png?filter=invert                    | 50    | 40     | 0,255,255   |
      | /img/50/colors.png?brightness=100                   | 50    | 40     | 255,255,255 |
      | /img/50/colors.png?brightness=10&contrast=20         | 50    | 40     |             |
      | /img/50/colors.png?filter=grayscale                 | 50    | 40     | gray        |
      | /img/10x40/colors.png?fit=cover&crop=top-left        | 10    | 40     | 255,0,0     |
      | /img/10x40/colors.png?fit=cover&crop=top-right       | 10    | 40     | 0,255,0     |

  Scenario: Canceled requests do not launch image converters
    Given a four-color image source
    When image URL "/img/50/colors.png" is requested after cancellation
    Then the image response status should be 500
