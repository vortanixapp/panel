# Security policy

## Reporting a vulnerability

Please do not open a public issue.

Report privately through GitHub's **Report a vulnerability** button under the
Security tab, or by email to **security@vortanix.app**.

Useful in a report: what an attacker gains, the steps to reproduce, and the
version or commit you tested. A working proof of concept is welcome but not
required — a clear description of the flaw is enough to start.

## What to expect

- Acknowledgement within three working days.
- An assessment, with our severity reading and whether we agree, within ten
  working days.
- Credit in the release notes, unless you would rather stay anonymous.

We do not run a paid bounty programme.

## Scope

This repository: the panel, the node agent, and the deployment recipes under
`deploy/`.

Out of scope: the hosted service at vortanix.app and its billing systems, which
are not part of this repository. Findings there are still welcome at the same
address.

## Supported versions

Until the first tagged release, only the default branch is supported. Fixes
land there and nowhere else.

## Please avoid

Testing against installations you do not own, denial of service, and social
engineering. Set up your own instance — it is a `docker compose up` away.
