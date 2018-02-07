# depot

Proxy program using socks5

# structure

```
--- client ---+------- server -------+------- local ------------------

client <--[socks5]--> depot-server <---> depot-local <--[localhost]--> app
```

# process

1. Server listens on the control port and socks port.
```
   +---+
   | C |
   +---+
   +---+
   | S |
   +---+
```

2. Local connects to the control port, and establish a control connection by
   handshaking.
   l->c: "hello server"
   c->l: "hello local"
```
   +---+
   | C |
   +---+
      +-+      handshake      +-+
      |c|<------------------->|l|
      +-+      [ctrlConn]     +-+
   +---+
   | S |
   +---+
```
   
3. Server listens on the tunnel port, and waits for socks request.
```
   +---+
   | C |
   +---+
      +-+                     +-+
      |c|<------------------->|l|
      +-+      [ctrlConn]     +-+
   +---+
   | S |
   +---+
   +---+
   | T |
   +---+
```
   
4. Client connects to the socks port and establish an socks connection after 
   handshaking and authenticating.
```
                           +---+
                           | C |
                           +---+
                              +-+                     +-+
                              |c|<------------------->|l|
                              +-+      [ctrlConn]     +-+
                           +---+
                           | S |
                           +---+
    +-+  handshake/auth   +-+
    |r|<----------------->|s|
    +-+   [socksConn]     +-+
                           +---+
                           | T |
                           +---+
```
 
5. Client sends the socks request to server. Server validates the request and 
   send it to local via control connection.
```
                           +---+
                           | C |
                           +---+
                              +-+   2.socks request   +-+
                              |c|<------------------->|l|
                              +-+     [ctrlConn]      +-+
                           +---+
                           | S |
                           +---+
    +-+  1.socks request  +-+
    |r|<----------------->|s|
    +-+   [socksConn]     +-+
                           +---+
                           | T |
                           +---+
```
	
6. Local parses the socks request and open an app connection to the target 
   host:port (i.e. app). After that, local sends success reply to server, and
   server sends reply to client.
```
                           +---+
                           | C |
                           +---+
                              +-+     2.reply         +-+
                              |c|<------------------->|l|
                              +-+     [ctrlConn]      +-+
                           +---+                      +-+   1.connect   +-+
                           | S |                      |b|<------------->|a|
                           +---+                      +-+   [appConn]   +-+
    +-+    3.reply        +-+
    |r|<----------------->|s|
    +-+   [socksConn]     +-+
                           +---+
                           | T |
                           +---+
```
   
7. Local connects to the tunnel port of server and establishs the tunnel
   connection by sending the socks request back as a handshake.
```
                           +---+
                           | C |
                           +---+
                              +-+       alive         +-+
                              |c|<------------------->|l|
                              +-+      [ctrlConn]     +-+
                           +---+                        
                           | S |                        
                           +---+                        
    +-+                   +-+ +-+     handshake     +-+ +-+                +-+
    |r|<----------------->|s|-|t|<----------------->|d|-|b|<-------------->|a|
    +-+   [socksConn]     +-+ +-+     [tunnelConn]  +-+ +-+    [appConn]   +-+
                           +---+
                           | T |
                           +---+
```

8. At this time, the pipeline is establish (like below). Client can communicate
    with app with socket r.

         socksConn <--> tunnelConn <--> appConn
		 
    The ctrlConn is used by local to send alive message. And once local exits,
	this connection would be closed. In the other hand, once server exits, local
	should close all connections and try to connect to control port of server 
	again and again.

# TODO

- [ ] local should try to connect to server repeatly and send heartbeat message
      to server after connecting.
- [ ] provide methods to watch the status of server and local.

