A simple tool for adding clients to Wireguard and printing QR codes of their configuration files

## Usage

Create `.env` file in the root of project with the following content:

```shell
SERVER_IP=<server_ip>
SERVER_PORT=<server_port>
DNS_IP=<dns_ip>
INTERFACE=wg0 (e.g.)
SERVER_PUBLIC_KEY_FILE=publickey (in /etc/wireguard/)
```

Prints the names of all clients:
```shell
# ./wg-add
```

Adds the client 'user' or outputs the QR code of its configuration file if it exists
```shell
# ./wg-add user
```