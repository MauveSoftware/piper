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
- prefix: 1.1.1.1/32
  source: 42
  target: 254
```

### Starting
```
./piper -config-file /path/to/config/file
```

## Third party libraries
* Netlink Go Library (https://github.com/vishvananda/netlink)

## License
(c) Mauve Mailorder Software GmbH & Co. KG, 2020. Licensed under [Apache 2.0](LICENSE) license.
