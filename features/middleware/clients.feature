Feature: Client identification and concurrent rate limiting
  Scenario Outline: Client addresses are normalized before rate limiting
    When client address "<address>" is normalized
    Then its IP should be "<ip>"

    Examples:
      | address       | ip          |
      | 127.0.0.1:123 | 127.0.0.1   |
      | [::1]:123     | ::1         |
      | ::1           | ::1         |
      | 192.168.1.1   | 192.168.1.1 |

  Scenario: Slow requests do not block each other inside the rate limiter
    Given a rate limiter allowing 10 requests per minute
    When two requests start slow downstream handlers
    Then both handlers should run concurrently
