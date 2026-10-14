#!/usr/bin/env python3
"""Send trace-rich Sentry error events to a Warnly ingest endpoint."""

import argparse
import json
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone
from urllib.parse import urlparse

DEFAULT_DSN = "https://pti2dj7@demo.warnly.io/ingest/3"


def envelope(dsn: str, number: int) -> bytes:
    parsed = urlparse(dsn)
    project_id = parsed.path.rstrip("/").rsplit("/", 1)[-1]
    key = parsed.username
    if not parsed.scheme or not parsed.hostname or not key or not project_id.isdigit():
        raise ValueError("DSN must look like https://PUBLIC_KEY@host/ingest/PROJECT_ID")

    event_id = uuid.uuid4().hex
    trace_id = uuid.uuid4().hex
    span_id = uuid.uuid4().hex[:16]
    parent_span_id = uuid.uuid4().hex[:16]
    now = datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")

    header = {
        "event_id": event_id,
        "sent_at": now,
        "dsn": dsn,
        "sdk": {"name": "warnly-demo-script", "version": "1.0.0"},
        "trace": {
            "environment": "production",
            "public_key": key,
            "release": "warnly-demo@1.0.0",
            "trace_id": trace_id,
        },
    }
    event = {
        "event_id": event_id,
        "timestamp": now,
        "platform": "python",
        "level": "error",
        "message": f"Demo traced failure #{number}: payment provider timed out",
        "environment": "production",
        "release": "warnly-demo@1.0.0",
        "server_name": "demo-client",
        "sdk": {"name": "warnly-demo-script", "version": "1.0.0"},
        "contexts": {
            "trace": {
                "trace_id": trace_id,
                "span_id": span_id,
                "parent_span_id": parent_span_id,
                "op": "http.server",
                "status": "internal_error",
                "description": "POST /api/checkout",
                "origin": "manual.demo",
                "data": {"http.method": "POST", "http.route": "/api/checkout", "http.status_code": 500},
            },
            "runtime": {"name": "CPython", "version": "3"},
        },
        "request": {"method": "POST", "url": "https://demo-client.local/api/checkout"},
        "tags": {"component": "checkout", "demo": "true"},
        "exception": [{
            "type": "TimeoutError",
            "value": "payment provider did not respond within 2 seconds",
            "stacktrace": {"frames": [
                {"function": "charge", "module": "payments.provider", "abs_path": "/app/payments/provider.py", "lineno": 84, "context_line": "return client.charge(order)", "in_app": True},
                {"function": "checkout", "module": "api.routes", "abs_path": "/app/api/routes.py", "lineno": 42, "context_line": "payment = charge(order)", "in_app": True},
            ]},
        }],
    }

    body = json.dumps(event, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    lines = [
        json.dumps(header, separators=(",", ":"), ensure_ascii=False).encode("utf-8"),
        json.dumps({"type": "event", "length": len(body)}, separators=(",", ":")).encode("ascii"),
        body,
    ]
    return b"\n".join(lines) + b"\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dsn", default=DEFAULT_DSN, help=f"Sentry DSN (default: {DEFAULT_DSN})")
    parser.add_argument("--count", type=int, default=1, help="number of distinct events to send")
    parser.add_argument("--delay", type=float, default=0.2, help="delay between events in seconds")
    args = parser.parse_args()
    if args.count < 1 or args.delay < 0:
        parser.error("--count must be positive and --delay cannot be negative")

    parsed = urlparse(args.dsn)
    project_id = parsed.path.rstrip("/").rsplit("/", 1)[-1]
    host = parsed.hostname
    if host is None:
        parser.error("DSN must include a hostname")
    # DSN userinfo contains the Sentry public key; it belongs in X-Sentry-Auth,
    # never in the request URL's authority section.
    authority = f"[{host}]" if ":" in host else host
    if parsed.port is not None:
        authority = f"{authority}:{parsed.port}"
    endpoint = f"{parsed.scheme}://{authority}/ingest/api/{project_id}/envelope/"
    auth = f"Sentry sentry_version=7, sentry_client=warnly-demo-script/1.0.0, sentry_key={parsed.username}"

    for i in range(1, args.count + 1):
        req = urllib.request.Request(
            endpoint,
            data=envelope(args.dsn, i),
            headers={"Content-Type": "application/x-sentry-envelope", "X-Sentry-Auth": auth},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=20) as response:
                print(f"[{i}/{args.count}] HTTP {response.status}: {response.read().decode('utf-8', 'replace')}")
        except urllib.error.HTTPError as exc:
            print(f"[{i}/{args.count}] HTTP {exc.code}: {exc.read().decode('utf-8', 'replace')}")
            return 1
        except urllib.error.URLError as exc:
            print(f"[{i}/{args.count}] request failed: {exc}")
            return 1
        if i < args.count and args.delay:
            time.sleep(args.delay)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
