This repository contains code for a prometheus service discovery on top of the
[SSLMate Cert Spotter][1]. The service discovery can be used to implement a
automatic certificate expiration monitoring using the prometheus
blackbox-exporter.

## Installation

The certspotter discovery can be installed by downloading the executable from
the [releases page][2] or by building it locally using make or docker.

```bash
make
# or
docker build -t certspotter-sd .
```

## Configuration

The certspotter service discovery can be configured using a configuration file
and command-line flags (configuration file to load and setting the logging
severity).

The configuration uses the following format.
```yaml
# global configuartion
global:
  # interval to use between polling the certspotter api.
  polling_interval: <duration>
  # rate limit to use for certspotter api (configured in Hz).
  rate_limit: <number>
  # token to used for authenticating againts certspotter api.
  token: <string>

# domains to query
domains:
    # domain to request certificate issuances for
  - domain: <string>
    # if sub domains should be included
    include_subdomains: <bool>
    
# files to export targets to
files:
    # filename to export targets to
  - file: <string>
    # labels to add to matching targets
    labels:
      <string>: <string>
    # target labels to match to be included in file
    match_re:
      <string>: <regex>
```

The certspotter service discovey is intended to be used with prometheus and the
blackbox-exporter this can be configured in prometheus as follows. A complete
configuration of certspotter-sd, blackbox-exporter and prometheus can be found
in the [example][3] folder.

```yaml
- job_name: "blackbox:tcp"
   metrics_path: /probe
   params:
     module: [tcp]
   file_sd_configs:
     - files:
         - /etc/prometheus/targets.json
       refresh_interval: 15s
   relabel_configs:
     - source_labels: [__address__, __port__]
       separator: ":"
       target_label: __param_target
     - source_labels: [__param_target]
       target_label: instance
     - target_label: __address__
       replacement: "localhost:9115"
```

Atm. configuration can't be reloaded by sending a `SIGHUP` and must be
terminated and restarted instead.

[1]: https://sslmate.com/certspotter/
[2]: https://github.com/codecentric/certspotter-sd/releases
[3]: https://github.com/codecentric/certspotter-sd/tree/master/example
