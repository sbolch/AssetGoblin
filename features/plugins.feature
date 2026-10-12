Feature: Runtime plugins
  As an AssetGoblin operator
  I want optional plugins to add and extend asset behavior at startup
  So that I can deploy capabilities without rebuilding the server

  Scenario: The example catalog composes audio, video and image extensions
    Given the "audio" example plugin is installed
    And the "video" example plugin is installed
    And the "qoi" example plugin is installed
    And the "qoa" example plugin is installed
    When the runtime plugins are loaded
    Then 2 plugin routes should be registered
    And the "qoi" image codec should be registered

  Scenario: An extension wraps its parent handler without duplicating the route
    Given a parent plugin serving "base" at "/custom/"
    And a child plugin extending the parent response with ":child"
    When the runtime plugins are loaded
    Then 1 plugin route should be registered
    When I send a GET request to "/custom/file"
    Then the response status should be 200
    And the response body should contain "base:child"

  Scenario: A missing plugin dependency rejects startup
    Given the "qoa" example plugin is installed
    When the runtime plugins are loaded
    Then plugin loading should fail with "missing parent plugin"

  Scenario: Plugin inheritance cycles reject startup
    Given a plugin that extends itself
    When the runtime plugins are loaded
    Then plugin loading should fail with "cyclic"

  Scenario: Independently deployed routes still require a valid signature
    Given a parent plugin serving "protected" at "/custom/"
    And the server requires the signing secret "test"
    When the runtime plugins are loaded
    And I send a GET request to "/custom/file"
    Then the response status should be 401

  Scenario: QOI output reuses core resizing and supports HEAD and byte ranges
    Given the image conversion tools are available
    And the "qoi" example plugin is installed
    And a PNG source "source.png" of size 100 by 80 exists
    When the runtime plugins are loaded
    And I send a GET request to "/img/50/source.qoi"
    Then the response status should be 200
    And the response should be a QOI image of size 50 by 40
    And the response content type should be "image/qoi"
    When I send a HEAD request to "/img/50/source.qoi"
    Then the response status should be 200
    And the response should have no body and a positive content length
    When I request bytes "0-3" from "/img/50/source.qoi"
    Then the response status should be 206
    And the response body should contain "qoif"

  Scenario: A QOI-only source can be resized into a native image format
    Given the image conversion tools are available
    And the "qoi" example plugin is installed
    And a PNG source "source.png" of size 100 by 80 exists
    When the runtime plugins are loaded
    And I send a GET request to "/img/50/source.qoi"
    Then the response status should be 200
    When I save the response as image source "decoded.qoi"
    And I send a GET request to "/img/25/decoded.png"
    Then the response status should be 200
    And the response should be a PNG image of size 25 by 20

  Scenario: The core does not offer QOI output without its plugin
    Given a PNG source "source.png" of size 100 by 80 exists
    When I send a GET request to "/img/50/source.qoi"
    Then the response status should be 400

  Scenario: The QOA extension trims and encodes through the audio parent
    Given the QOA conversion tools are available
    And the "audio" example plugin is installed
    And the "qoa" example plugin is installed
    And a one-second WAV audio source exists
    When the runtime plugins are loaded
    Then 1 plugin route should be registered
    When I send a GET request to "/a/source.wav?duration=0.5&format=qoa&volume=0.5"
    Then the response status should be 200
    And the response should be QOA audio with 24000 samples
    And the response content type should be "audio/qoa"

  Scenario: QOA input can be decoded or trimmed and encoded again
    Given the QOA conversion tools are available
    And the "audio" example plugin is installed
    And the "qoa" example plugin is installed
    And a one-second WAV audio source exists
    When the runtime plugins are loaded
    And I send a GET request to "/a/source.wav?duration=0.5&format=qoa"
    Then the response status should be 200
    When I save the response as audio source "source.qoa"
    And I send a GET request to "/a/source.qoa?duration=0.25&format=wav"
    Then the response status should be 200
    And the response body should contain "RIFF"
    When I send a GET request to "/a/source.qoa?duration=0.25&format=qoa"
    Then the response status should be 200
    And the response should be QOA audio with 12000 samples
    When I request bytes "0-3" from "/a/source.qoa"
    Then the response status should be 206
    And the response body should contain "qoaf"

  Scenario: Extending audio preserves its existing MP3 conversion
    Given the QOA conversion tools are available
    And the "audio" example plugin is installed
    And the "qoa" example plugin is installed
    And a one-second WAV audio source exists
    When the runtime plugins are loaded
    And I send a GET request to "/a/source.wav?duration=0.2&format=mp3"
    Then the response status should be 200
    And the response body should not be empty

  Scenario: QOA output preserves the parent's required duration validation
    Given the "audio" example plugin is installed
    And the "qoa" example plugin is installed
    And an audio source "source.wav" exists
    When the runtime plugins are loaded
    And I send a GET request to "/a/source.wav?format=qoa"
    Then the response status should be 400
    And the response body should contain "require duration"

  Scenario: A valid signature grants access to an independently deployed route
    Given a parent plugin serving "protected" at "/custom/"
    And the server requires the signing secret "test"
    When the runtime plugins are loaded
    And I request "/custom/file" with a valid server signature
    Then the response status should be 200
    And the response body should contain "protected"

  Scenario: Plugin routes cannot replace the core image route
    Given a parent plugin serving "conflict" at "/custom/"
    And the image route prefix is "/custom/"
    When the runtime plugins are loaded
    And the server is configured
    Then server configuration should fail with "duplicate route"
