Feature: Audio and video plugins
  As an AssetGoblin operator
  I want scripts to define lightweight media transformations
  So that installing a plugin adds useful behavior without media logic in core

  Scenario Outline: Audio trimming and conversion use the requested codec
    Given the "audio" example plugin is installed
    And a one-second "audio" media source exists
    When the runtime plugins are loaded
    And I send a GET request to "/a/source.wav?start=0.2&duration=0.5&volume=0.5&format=<format>"
    Then the response status should be 200
    And the response should use codec "<codec>" with a half-second duration
    When I request bytes "0-9" from "/a/source.wav?start=0.2&duration=0.5&volume=0.5&format=<format>"
    Then the response status should be 206
    And the response body should have length 10
    When I send a HEAD request to "/a/source.wav?start=0.2&duration=0.5&volume=0.5&format=<format>"
    Then the response status should be 200
    And the response should have no body and a positive content length

    Examples:
      | format | codec     |
      | mp3    | mp3       |
      | wav    | pcm_s16le |
      | ogg    | vorbis    |

  Scenario Outline: Video cropping and resizing precede the requested conversion
    Given the "video" example plugin is installed
    And a one-second "video" media source exists
    When the runtime plugins are loaded
    And I send a GET request to "/v/source.mp4?duration=0.5&crop=200:100:10:10&width=100&height=50&format=<format>"
    Then the response status should be 200
    And the response should use codec "<codec>" with a half-second duration
    And the video response should have size 100 by 50
    When I request bytes "0-9" from "/v/source.mp4?duration=0.5&crop=200:100:10:10&width=100&height=50&format=<format>"
    Then the response status should be 206
    And the response body should have length 10
    When I send a HEAD request to "/v/source.mp4?duration=0.5&crop=200:100:10:10&width=100&height=50&format=<format>"
    Then the response status should be 200
    And the response should have no body and a positive content length

    Examples:
      | format | codec |
      | mp4    | h264  |
      | webm   | vp8   |

  Scenario Outline: Audio scripts reject invalid transformation parameters
    Given the "audio" example plugin is installed
    And an audio source "source.wav" exists
    When the runtime plugins are loaded
    And I send a GET request to "<request>"
    Then the response status should be 400

    Examples:
      | request                                 |
      | /a/source.wav?format=mp3                |
      | /a/source.wav?duration=61               |
      | /a/source.wav?duration=1&format=invalid |
      | /a/source.wav?duration=1&start=oops     |

  Scenario Outline: Video scripts reject invalid transformation parameters
    Given the "video" example plugin is installed
    And a video source "source.mp4" exists
    When the runtime plugins are loaded
    And I send a GET request to "/v/source.mp4?<query>"
    Then the response status should be 400

    Examples:
      | query                     |
      | format=mp4                |
      | duration=61               |
      | duration=1&format=invalid |
      | duration=1&start=oops     |
