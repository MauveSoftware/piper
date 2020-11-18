[![CircleCI](https://circleci.com/gh/MauveSoftware/piper.svg?style=shield)](https://circleci.com/gh/MauveSoftware/piper)
[![Go Report Card](https://goreportcard.com/badge/github.com/mauvesoftware/piper)](https://goreportcard.com/report/github.com/mauvesoftware/piper)

# piper
Pipe routing information from one routing table to another using netlink

## Install

### via go get
```bash
go get github.com/MauveSoftware/piper
```

## Configuration
If we want to pipe the prefix 1.1.1.1/32 learned in table 42 to the main table:

```yaml
proto: 188 
pipes:
- name: "dns"
  prefix: 1.1.1.1/32
  source: 42
  target: 254
```

### Starting
```
./piper -config-file /path/to/config/file
```

## Metrics
Piper provides a prometheus metrics endpoint on default port :10080

## Third party libraries
* Netlink Go Library (https://github.com/vishvananda/netlink)

## License
(c) Mauve Mailorder Software GmbH & Co. KG, 2020. Licensed under [Apache 2.0](LICENSE) license.
