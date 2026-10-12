Feature: Middleware
  As an API consumer
  I want rate limiting and signature verification middleware to work correctly
  So that my server is protected from abuse

  Scenario: Valid signature allows access
    Given a signkey middleware with secret "mysecret"
    When a valid token is generated for path "/img/thumbnail/test.jpg"
    And I send a GET request to "/img/thumbnail/test.jpg" with that token
    Then the response status should be 200

  Scenario: Invalid signature is rejected
    Given a signkey middleware with secret "mysecret"
    When I send a GET request to "/img/thumbnail/test.jpg" with token "invalidtoken"
    Then the response status should be 401

  Scenario: Missing signature is rejected
    Given a signkey middleware with secret "mysecret"
    When I send a GET request to "/img/thumbnail/test.jpg" without a token
    Then the response status should be 401

  Scenario: Rate limit allows requests within limit
    Given a rate limit middleware configured for 10 requests per minute
    When I send 5 requests within the time window
    Then all requests should be successful

  Scenario: Rate limit blocks excess requests
    Given a rate limit middleware configured for 3 requests per minute
    When I send 4 requests in rapid succession
    Then the 4th request should be rate limited

  Scenario: Rate limit resets after time window
    Given a rate limit middleware configured for 2 requests with a 100ms window
    When I send 2 requests within the window
    And I wait for the window to expire
    And I send another request
    Then that request should be successful

  Scenario Outline: Signatures are bound to the secret and request path
    Given a signkey middleware with secret "secret"
    And a token generated with secret "<secret>" for path "<signed_path>"
    When I send a GET request to "/test-path" with that token
    Then the response status should be <status>

    Examples:
      | secret       | signed_path | status |
      | secret       | /test-path  | 200    |
      | wrong-secret | /test-path  | 401    |
      | secret       | /wrong-path | 401    |
