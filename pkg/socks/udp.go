package socks

// TODO: implement a UDP relay server that can be started and stopped (with ctx) on an ASSOCIATE command
// SOCKS needs a separate relay stream for the connection.. Tunnel the UDP traffic through a yamux stream.
// Inspiration:
// https://github.com/txthinking/socks5/tree/master
// https://stackoverflow.com/questions/62283351/how-to-use-socks-5-proxy-with-tidudpclient-properly
