```
                _
   ____  ____  (_)___  ___
  / __ \/ __ \/ / __ \/ _ \
 / / / / /_/ / / /_/ /  __/
/_/ /_/ .___/_/ .___/\___/
     /_/     /_/
```

https://npipe.io/

Share files between devices on the same network.

## Why?

Because it's the year 2026, the future is here with computers creating art, but
I still used Signal to share files between devices next to each other. So I had
futuristic technology build me something simple. It was my first time trying out
Claude, and I think it did pretty well!

## How?

The server puts clients into different subnets per [RFC
1918](https://www.rfc-editor.org/rfc/rfc1918.html):

* 10.0.0.0/8
* 172.16.0.0/12
* 192.168.0.0/16

Clients can download files when they connect from the same subnet as the
uploader.

A trusted reverse proxy makes sure that the source IP address is set based on
the TCP connection. Attackers can spoof this address, but to my knowledge they
won't receive the response.

/32 (IPv4) or /128 (IPv6) is used for clients that connect with a public IP
address. The server assumes that they own their IP address. However, CG-NAT
breaks that assumption: ISPs assign the same public IP address to multiple
customers. The same happens for VPN users sharing exit nodes.

This makes using my hosted instance, https://npipe.io/, pretty bad for privacy
from other users, without even considering privacy from whoever runs the server
(me), so I don't recommend actually using it, but hosting it yourself.

---

<sub>inspired by [ncat --broker](https://man.archlinux.org/man/ncat.1.en)</sub>
