# modware-user
<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-1-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->

[![License](https://img.shields.io/badge/License-BSD%202--Clause-blue.svg)](LICENSE)  
![Continuous integration](https://github.com/dictyBase/modware-user/workflows/Continuous%20integration/badge.svg)
[![codecov](https://codecov.io/gh/dictyBase/modware-user/branch/develop/graph/badge.svg)](https://codecov.io/gh/dictyBase/modware-user)
[![Maintainability](https://api.codeclimate.com/v1/badges/30e9b0421a28b8e0d941/maintainability)](https://codeclimate.com/github/dictyBase/modware-user/maintainability)  
![Last commit](https://badgen.net/github/last-commit/dictyBase/modware-user/develop)   
[![Funding](https://badgen.net/badge/Funding/Rex%20L%20Chisholm,dictyBase,DCR/yellow?list=|)](https://projectreporter.nih.gov/project_info_description.cfm?aid=10024726&icde=0)

[dictyBase](http://dictybase.org) **API** server for managing users, their
roles and permissions. The API server supports both gRPC and HTTP/JSON protocol
for data exchange.

## Usage

```
NAME:
   modware-user - starts the modware-user microservice with HTTP and grpc backends

USAGE:
   modware-user [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dictyuser-pass value  dictyuser database password [$DICTYUSER_PASS]
   --dictyuser-db value    dictyuser database name [$DICTYUSER_DB]
   --dictyuser-user value  dictyuser database user [$DICTYUSER_USER]
   --dictyuser-host value  dictyuser database host (default: "dictyuser-backend") [$DICTYUSER_BACKEND_SERVICE_HOST]
   --dictyuser-port value  dictyuser database port [$DICTYUSER_BACKEND_SERVICE_PORT]
   --port value            tcp port at which the servers will be available (default: "9596")
   --help, -h              show help
   --version, -v           print the version

```

## API

### gRPC

The protocol buffer definitions and service apis are documented
[here](https://github.com/dictyBase/dictybaseapis/tree/master/dictybase/user).

# Misc. badges
![Issues](https://badgen.net/github/issues/dictyBase/modware-user)
![Open Issues](https://badgen.net/github/open-issues/dictyBase/modware-user)
![Closed Issues](https://badgen.net/github/closed-issues/dictyBase/modware-user)  
![Total PRS](https://badgen.net/github/prs/dictyBase/modware-user)
![Open PRS](https://badgen.net/github/open-prs/dictyBase/modware-user)
![Closed PRS](https://badgen.net/github/closed-prs/dictyBase/modware-user)
![Merged PRS](https://badgen.net/github/merged-prs/dictyBase/modware-user)  
![Commits](https://badgen.net/github/commits/dictyBase/modware-user/develop)
![Branches](https://badgen.net/github/branches/dictyBase/modware-user)
![Tags](https://badgen.net/github/tags/dictyBase/modware-user/?color=cyan)  
![GitHub repo size](https://img.shields.io/github/repo-size/dictyBase/modware-user?style=plastic)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/dictyBase/modware-user?style=plastic)
[![Lines of Code](https://badgen.net/codeclimate/loc/dictyBase/modware-user)](https://codeclimate.com/github/dictyBase/modware-user/code)  

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="http://cybersiddhu.github.com/"><img src="https://avatars3.githubusercontent.com/u/48740?v=4" width="100px;" alt=""/><br /><sub><b>Siddhartha Basu</b></sub></a><br /><a href="https://github.com/dictyBase/modware-user/issues?q=author%3Acybersiddhu" title="Bug reports">🐛</a> <a href="https://github.com/dictyBase/modware-user/commits?author=cybersiddhu" title="Code">💻</a> <a href="#content-cybersiddhu" title="Content">🖋</a> <a href="https://github.com/dictyBase/modware-user/commits?author=cybersiddhu" title="Documentation">📖</a> <a href="#maintenance-cybersiddhu" title="Maintenance">🚧</a></td>
  </tr>
</table>

<!-- markdownlint-enable -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!