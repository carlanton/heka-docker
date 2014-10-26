# Deprecated!

This plugin has been merged into Heka core under the name DockerLogInput. Just use that instead :-)


heka-docker
===========

Docker input plugin for [Mozilla Heka](https://hekad.readthedocs.org). Heavily
based on [progrium/logspout](https://github.com/progrium/logspout) and
[victorcoder/heka-redis](https://github.com/victorcoder/heka-redis).

The plugin connects to the Docker socket, pulls all log messages (stdout/stderr)
from all running containers and converts them into Heka messages.

## Usage

See [Building *hekad* with External
Plugins](http://hekad.readthedocs.org/en/latest/installing.html#build-include-externals)
for compiling in plugins.

```toml
[DockerInput]
endpoint = "unix:///var/run/docker.sock"

[ESJsonEncoder]

[debug]
type = "LogOutput"
message_matcher = "Type == 'docker-container'"
encoder = "ESJsonEncoder"
```
