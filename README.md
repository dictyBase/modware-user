# modware-user
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

### HTTP/JSON
