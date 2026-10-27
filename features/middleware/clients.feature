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

  Scenario Outline: Ports and forwarding headers do not bypass per-IP limits
    Given a rate limiter allowing 2 requests per minute
    When a request from "<first>" returns 200 with 1 remaining
    And a request from "<second>" returns 200 with 0 remaining
    And a request from "<third>" returns 429 with 0 remaining
    Then the response includes rate limit and retry headers
    And a request from "203.0.113.99:123" returns 200 with 1 remaining

    Examples:
      | first           | second          | third           |
      | 192.0.2.1:123   | 192.0.2.1:456   | 192.0.2.1       |
      | [2001:db8::1]:1 | [2001:db8::1]:2 | 2001:db8::1     |

  Scenario: Accepted requests do not extend a fixed window
    When accepted requests keep the original window reset

  Scenario: Closing the limiter releases its store and is repeatable
    Given a rate limiter allowing 2 requests per minute
    When the limiter is closed twice
    Then the closed limiter rejects new requests

  Scenario Outline: Invalid active limiter configurations are rejected
    Then a limiter with limit <limit> and TTL "<ttl>" is rejected

    Examples:
      | limit | ttl |
      | 0     | 1m  |
      | -1    | 1m  |
      | 1     | 0s  |
      | 1     | -1s |
