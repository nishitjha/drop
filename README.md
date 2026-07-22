
# Drop

A cross-platform, peer-to-peer file (and directory/text snippet) sharing CLI. It discovers other devices on your local network with zero configuration mDNS and allows direct sharing; the only limit being your own disk speed and bandwidth.

It also comes with a daemon that can (optionally) be installed by the user to listen to sharing requests and receive files in the background 24/7. This feature, I believe, is mainly what led to me creating this as opposed to using a pre-existing utility as they mostly require you to have both devices open for the transfer. They say it's for "security" - like I'm gonna send myself a ZIP bomb from another device.

## Features

- **Zero-config discovery**
- **Support for folders and text snippets**
- **Interactive or scriptable** — pick a device and file with an interactive picker, or pass everything as arguments for use in scripts.
- **Trust system**
- **Background daemon mode** — install Drop as a system service so your machine is always discoverable and can receive files without a terminal open.

## Requirements
The only requirement is **a network that allows multicast DNS traffic between devices**. Usually, large public networks (say, in cafés, hotels or airports) do not support said traffic.

Actually, I lied - there are some other requirements. **Device discovery and broadcasting is also contingent on the Drop executable being allowed to do so by the system firewall**. In case you run into unintended behaviour, check if your firewall is the culprit.

Lastly, you may sometimes run into problems streaming files or folders when you haven't given Drop read access to said files/folders. Save yourself the trouble and just give it full disk access.
## Installation
You can either install Drop with go or get the pre-built binaries.

#### Use 'go install'
```bash
  go install https://github.com/nishitjha/drop@latest
```
This puts the `drop` binary in `$(go env GOPATH)/bin` — make sure that directory is on your `PATH`.

## Usage

### Help in the terminal

```bash
drop --help
# or just: drop help
```
 
**Help for a specific command**: to learn about its description, aliases, and flags:
 
```bash
drop share --help
drop config --help
drop service --help
# or just use drop help <command>
```

### List devices on your network
 
```bash
drop list
# aliases: drop ls | drop devices | drop peers
```
 
Shows every discoverable Drop instance on your network, its status, and its IPV4 addresss.
 
### Sharing
```bash
drop share <device_name> <path>
# aliases: drop send | drop sh
```
 
- If you omit `<path>`, an interactive file picker opens.
- To share a **folder** instead of a single file, add `--dir` / `-d`:
```bash
  drop share -d <device_name> <folder_path>
```
- To share **plain text** instead of files or folders:
```bash
  drop share -t <device_name> "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```
 
The receiving device gets a 3-minute window to accept or decline your request, either via a terminal prompt (in interactive mode) or a browser page (in daemon mode).
 
### View or change configuration
 
```bash
drop config                          # list every setting, its current value, and a description
drop config <setting>                # show one setting
drop config <setting> <new_value>    # change a setting
```
 
### Service management

As opposed to running `drop` in the terminal whenever you know you're going to receive a request, you can also run Drop persistently in the background so your machine stays discoverable and can receive files without a terminal open.
 
```bash
drop service install   # or: drop service i   — install and start as a background service
drop service start     # or: drop service s
drop service kill       # or: drop service k
drop service uninstall  # or: drop service u
```

Currently, requests from an untrusted device lead to a request page opening up in the browser, though I will later implement native prompts (but they look hideous not gonna lie).
## Contributing

Very much open to contributions, no code of conduct, no `contributing.md` - knock yourself out.
## License

MIT
