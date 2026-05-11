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

## How this works

Devices behind the same router share a public IP address
([NAT](https://en.wikipedia.org/wiki/Network_address_translation)). The server
uses this to enable file sharing between them: devices can download files when
they have the same source IP address as the uploader.

I'd consider this abuse of NAT, and not how the internet should be used, but it
works in most cases.

## Is this safe?

**If you mean privacy from others:**

Not sure. A trusted reverse proxy makes sure that the source IP address is set
based on the TCP connection. Attackers can spoof this address, but according to
my knowledge, they will not receive the response.

As far as I can tell, the biggest vulnerability here is CG-NAT. ISPs assign the
same public IP address to multiple customers (with different ports). So if two
customers with the same IP address happen to visit npipe.io, they will be able
to download each other's files.

VPN customers with the same exit node will encounter the same problem.

**If you mean privacy from me:**

Files are never written to disk and are automatically deleted after 5 minutes.
However, don't trust me with your incriminating evidence. As far as you know,
this is a CIA operation targeting you.

## Is this illegal?

Probably
