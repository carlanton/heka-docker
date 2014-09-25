heka-docker
===========

Docker input plugin for [Mozilla Heka](https://hekad.readthedocs.org). Heavily
based on [progrium/logspout](https://github.com/progrium/logspout) and
and [victorcoder/heka-redis](https://github.com/victorcoder/heka-redis).

The plugin connect to the Docker socket, pull all log messages (stdout/stderr)
from all running containers and convert them into Heka messages.

## Usage

See [Building *hekad* with External
Plugins](http://hekad.readthedocs.org/en/latest/installing.html#build-include-externals)
for compiling in plugins.

```toml
[DockerInput]

[ESJsonEncoder]

[debug]
type = "LogOutput"
message_matcher = "Type == 'docker_log'"
encoder = "ESJsonEncoder"
```
