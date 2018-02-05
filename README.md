# depot

Proxy program using socks5

# structure

```
--- client ----+------- server --------+------- local ------------------

client <--[socks5]--> depot-server <---> depot-local <--[localhost]--> app
```

# TODO

- [ ] local should try to connect to server repeatly and send heartbeat message
      to server after connecting.
- [ ] provide methods to watch the status of server and local.
