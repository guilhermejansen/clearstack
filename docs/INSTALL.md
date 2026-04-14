# Installing clearstack

clearstack ships as a single static Go binary. Pick whichever channel you
trust the most.

## macOS / Linux

### Homebrew

```sh
brew install guilhermejansen/tap/clearstack
```

### Install script

```sh
curl -fsSL https://raw.githubusercontent.com/guilhermejansen/clearstack/main/install.sh | bash
```

Env vars:

- `CLEARSTACK_INSTALL_DIR` — override default `$HOME/.local/bin`
- `CLEARSTACK_VERSION` — pin a specific version, e.g. `CLEARSTACK_VERSION=v1.0.0`

### Direct binary

Grab the archive matching your OS/arch from the
[latest GitHub release](https://github.com/guilhermejansen/clearstack/releases/latest),
extract it, and drop `clearstack` somewhere on `$PATH`.

## Windows

### Scoop

```powershell
scoop bucket add guilhermejansen https://github.com/guilhermejansen/scoop-bucket
scoop install clearstack
```

### Winget

```powershell
winget install GuilhermeJansen.clearstack
```

### Direct

Download the `.zip` matching your architecture from the releases page,
extract `clearstack.exe`, and add its directory to `PATH`.

## From source

Requires Go 1.24+.

```sh
git clone https://github.com/guilhermejansen/clearstack.git
cd clearstack
make build          # builds ./bin/clearstack
# or
go install github.com/guilhermejansen/clearstack/cmd/clearstack@latest
```

## Verifying the install

```sh
clearstack version
clearstack doctor
```

`doctor` should report the OS, config/state paths, trash backend, and the
list of categories available given the tools on your `$PATH`.
