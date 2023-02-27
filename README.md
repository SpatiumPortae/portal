# Portal

<p align="center">
<img src="https://user-images.githubusercontent.com/6842167/172497072-e196c2d0-f0f9-4039-83f4-5d7e056e97cf.png" width="375" height="auto">
</p>
<p align="center" style="font-weight: bold;">
a command-line file transfer utility for sending files from any computer to another
</p>
  
<br>

[![Build status](https://img.shields.io/github/actions/workflow/status/SpatiumPortae/portal/ci.yml?branch=master)](https://img.shields.io/github/actions/workflow/status/SpatiumPortae/portal/ci.yml?branch=master)


## Installation

On any platform, you can get the [latest release manually](https://github.com/SpatiumPortae/portal/releases/latest), or simply run:

```bash
curl -s https://raw.githubusercontent.com/SpatiumPortae/portal/master/scripts/install.sh | bash
```

On MacOS or Linux, if you are using Homebrew:
```bash
brew install SpatiumPortae/homebrew-portal/portal
```

## How it works

### Sending files and folders

To send files:

```bash
portal send <file1> <file2> <folder1> <folder2> ...
```

The application will output a temporary password on the format `1-inertia-elliptical-celestial`.
<br>
The sender will communicate this password to the receiver over some secure channel.

### Receiving files and folders

To receive those files:

```bash
portal receive 1-intertia-elliptical-celestial
```

The two clients will establish a connection through a relay server. The file transfer will then commence with a direct or relayed connection, depending on which one is available.

### Demo

![demo](./assets/demo.gif)

## Features

`portal` provides:

- Hosting your own relay (we'd appreciate it if you plan to send a lot of data!)
- Changing the default configuration to your liking (see [link to config])
- End-to-end encryption using [PAKE2](https://en.wikipedia.org/wiki/Password-authenticated_key_agreement)
- Direct transfer of files if possible (e.g. sender and receiver are in the same local network)
- Fallback to a relay server for file transfer if the sender and receiver cannot connect directly
- Parallel gzip compression of files for faster and more efficient transfers

<details>
  <summary>Technical details</summary>
  
## Technical details

The connection between the sender and the server is negotiated using a intermediary server (relay).
<br>
The relay server is used to negotiate a secure encrypted channel while never seeing the contents of files nor the temporary password.

The communication works as follows:

- `sender` connects to `relay`
- `relay` allocates an id to the sender and sends it to the `sender`
- `sender` outputs the password to the terminal, hashes the password and sends it to the `relay`
- `receiver` hashes the password (which has been communicated over some secure channel) and sends it to the `relay`
- When both the `sender` and the `receiver` have sent the hashed password to the `relay`, the cryptographic exchange starts
- During the cryptographic exchange, the `relay`, well, relays messages from the `sender` to the `receiver` and vice-versa
- Once the cryptographic exchange is done, every message sent by the `sender` and `receiver` is encrypted, and the `relay` cannot see their contents
- The file transfer is about to begin, and can commence in two ways: 
  1. The `sender` and `receiver` are in the same local network or can be reached directly by IP in some other way
    - In this case, the `sender` and `receiver` will happily send the files to each other directly. The `relay` will close down for this connection.
  2. The `sender` and `receiver` are not on the same local network, or cannot reach each other directly. The transfer will go through the `relay`, which will continue to relay encrypted messages until the file transfer is completed
 </details>

## Maintainers

- [Arvid Gotthard](https://github.com/mellonnen)
- [Zino Kader](https://github.com/ZinoKader)

## Possible thanks to...

[nhooyr/websocket](https://github.com/nhooyr/websocket), [shollz/pake](https://github.com/schollz/pake), [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles), [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea), [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss), [muesli/reflow](https://github.com/muesli/reflow), [klauspost/pgzip](https://github.com/klauspost/pgzip) and many, many more.
