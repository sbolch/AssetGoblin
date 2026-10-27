---
title: Request security
weight: 4
url: /docs/security.html
---

# Request security

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/security.png">
    <img src="../assets/mascot/security.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

## Rate limiting

The rate limiter uses go-limiter's in-memory store to count requests per client IP
in fixed windows. Each IP's first request starts its window; the allowance resets
at each configured interval, even when requests continue arriving. All paths share
the same per-IP allowance. The client address comes from the socket, ignoring
forwarding headers and port changes.

Responses include `X-RateLimit-Limit`, `X-RateLimit-Remaining` and
`X-RateLimit-Reset`. Requests exceeding the allowance return HTTP 429 with
`Retry-After`. Reset and retry times use HTTP dates in GMT. Limits are local to
each server process. The server closes the store and its cleanup worker on shutdown.

Configure `rate_limit.limit` as the maximum accepted request count and
`rate_limit.ttl` as the positive window duration, using Go's `time.ParseDuration`
syntax (for example `1m`). Set `limit` to `0` to disable rate limiting.

## Signed URLs

The token is a hash string that is used to validate the request.
It's generated from a secret and the request path and is used to prevent unauthorized access to the server.

The secret is defined in the config file under the `secret` key. If you don't want to use tokens, just leave it empty.

If you use it, you must use the same secret to generate the token in your requests.
You have to pass the token as get parameter in the URL.

You can generate the token like this:

In Go:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func generateToken(secret, path string) string {
    hasher := hmac.New(sha256.New, []byte(secret))
    hasher.Write([]byte(path))
    return hex.EncodeToString(hasher.Sum(nil))
}
```

In PHP:

```php
$token = hash_hmac('sha256', $path, $secret);
```

Other languages can use the same HMAC-SHA256 algorithm.
