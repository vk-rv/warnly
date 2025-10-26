<h1>
<p align="center">
<br>Warnly
</h1>
  <p align="center">
    Exception monitoring system, designed specifically for self-hosting
    <br />
    <a href="#about">About</a>
    ·
    <a href="#demo">Demo</a>
    ·
    <a href="#documentation">Documentation</a>
    ·
    <a href="internal#project-structure">Developing</a>
  </p>
</p>

## About

Error logs should be categorized into issues, with each issue assigned to the appropriate team member. In an ideal scenario, a well-functioning application should operate silently. Warnly, in line with Sentry's best practices, address this effectively.

Enterprise-focused solutions tend to prioritize complex features that create unnecessary overhead for self-hosting scenarios. There's an opportunity to take the core monitoring functionality and package it into a single binary that eliminates operational complexity while maintaining essential features. That's how Warnly was born: an open-source project designed specifically for self-hosting.

For more details, see [About Warnly](https://docs.warnly.io/).

## Demo

Try the demo application at [https://demo.warnly.io](https://demo.warnly.io). Use username `admin` and password `admin` to sign in. The infrastructure is graciously provided by [VPSDime](https://vpsdime.com/).

## Documentation

See the [documentation](https://docs.warnly.io/) on the Warnly website.

## Roadmap and Status

The high-level plan for `warnly`, in order:

|  #  | Step                                                      | Status |
| :-: | --------------------------------------------------------- | :----: |
|  1  | Backend exception monitoring                              |   ⚠️   |
|  2  | Mobile and frontend exception monitoring                  |   ❌   |
|  3  | Swappable storage                                         |   ❌   |
|  4  | SLO and flexible alerting rules                           |   ❌   |
|  N  | Fancy features (to be expanded upon later)                |   ❌   |
