Feature: Server Behavior
  As an AssetGoblin user
  I want the HTTP server to serve assets correctly
  So that I can access static files and images

  Scenario: Server serves static files from public directory
    Given a configured server with public directory "testdata/public"
    And a static file "hello.txt" exists with content "Hello, World!"
    When I send a GET request to "/hello.txt"
    Then the response status should be 200
    And the response body should contain "Hello, World!"

  Scenario: Server returns 404 for non-existent file
    Given a configured server with public directory "testdata/public"
    When I send a GET request to "/nonexistent.html"
    Then the response status should be 404

  Scenario: Server returns bad request for unsupported image format
    Given a configured server with image preset "thumbnail" of width 100
    And a configured server with supported image formats "jpg" and "png"
    And an image file "testdata/assets/img/test.jpg" exists
    When I send a GET request to "/img/thumbnail/test.gif"
    Then the response status should be 400

  Scenario: Server rejects short image path
    Given a configured server
    When I send a GET request to "/img/short"
    Then the response status should be 404

  Scenario Outline: Asset routes support GET and HEAD and reject other methods
    Given a static file "hello.txt" exists with content "hello"
    And the image route prefix is "/media/images/"
    When I send a "<method>" request to "<path>"
    Then the response status should be <status>

    Examples:
      | method | path                              | status |
      | GET    | /hello.txt                        | 200    |
      | HEAD   | /hello.txt                        | 200    |
      | POST   | /hello.txt                        | 405    |
      | GET    | /media/images/640/missing.png     | 404    |
      | GET    | /media/images/640/missing.gif     | 400    |
      | GET    | /media/images/invalid/missing.png | 400    |

  Scenario: Signing also protects static files
    Given a static file "hello.txt" exists with content "hello"
    And the server requires the signing secret "test"
    When I send a GET request to "/hello.txt"
    Then the response status should be 401

  Scenario Outline: Invalid image route prefixes reject server configuration
    Given the image route prefix is "<prefix>"
    When the server is configured
    Then server configuration should fail with "invalid or duplicate route"

    Examples:
      | prefix   |
      | /        |
      | img/     |
      | /img     |
      | /{img}/  |
      | /bad%/   |
      | /a//b/   |
      | /a/../b/ |

  Scenario: The production listener applies middleware and stops on cancellation
    Given the server requires the signing secret "test"
    And the server allows 1 requests per minute
    When the production server starts listening
    And I send a GET request to "/missing"
    Then the response status should be 401
    When I send a GET request to "/missing"
    Then the response status should be 429
    When the production server is stopped
