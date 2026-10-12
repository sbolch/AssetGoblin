Feature: Image Processing
  As a user
  I want images to be processed with presets and direct sizing
  So that I can get appropriately sized images

  Scenario: Request with invalid preset returns error
    Given an image service configured with preset "thumbnail" of width 50
    And a source image "test.jpg" exists in the image directory
    When I request an image with path "/img/invalid/test.jpg"
    Then the response status should be 400

  Scenario: Request with non-existent source file returns 404
    Given an image service configured with preset "thumbnail" of width 50
    When I request an image with path "/img/thumbnail/missing.jpg"
    Then the response status should be 404

  Scenario: Request with short path returns 404
    Given an image service configured with preset "thumbnail" of width 50
    When I request an image with path "/img/short"
    Then the response status should be 404

  Scenario: Request with unsupported format returns 400
    Given an image service configured with supported formats "jpg" and "png"
    When I request an image with path "/img/thumbnail/test.gif"
    Then the response status should be 400

  Scenario: Request with valid preset and source file
    Given an image service configured with preset "thumbnail" of width 50
    And a source image "test.jpg" exists in the image directory
    When I request an image with path "/img/thumbnail/test.jpg"
    Then the image response should be successful

  Scenario Outline: Image requests reject traversal before touching the source directory
    Given the image route prefix is "/nested/img/"
    When I request an image with path "<path>"
    Then the response status should be 404

    Examples:
      | path                             |
      | /nested/img/100/../secret.jpg    |
      | /nested/img/100//secret.jpg      |
      | /nested/img/100/../../secret.jpg |
      | /other/img/100/test.jpg          |

  Scenario: Core resizing writes to an absolute cache directory
    Given the core image converter is available
    And a PNG source "source.png" of size 100 by 80 exists
    When I send a GET request to "/img/50/source.png"
    Then the response status should be 200
    And the response should be a PNG image of size 50 by 40
    And the image cache should contain the transformed image
