# Vrsr (VeRSioneR)

Vrsr is a lightweight CLI tool for managing multiple installed versions of developer binaries. It makes it easy to download, install and switch between versions of tools (for example, tools distributed as GitHub releases or simple downloadable archives), keeping separate copies per-version and exposing the active version via a symbolic link to version chosen by the user.

## Installation

- Install from source (requires Go):

```sh
go install github.com/stepbeta/vrsr@latest
```

- Download a prebuilt binary from the GitHub Releases page:

```sh
# visit https://github.com/stepbeta/vrsr/releases (replace version accordingly):
curl -L "https://github.com/stepbeta/vrsr/releases/download/vX.Y.Z/vrsr"
sudo mv vrsr /usr/local/bin/
```

After installation, configure the paths used by `vrsr` (or use the defaults) via flags or `viper` configuration. Typical settings you may want to set are the `vrs-path` (where downloaded versions are stored) and `bin-path` (where the active symlink is created).

## How to use

Vrsr manages a small set of developer tools via per-tool subcommands. The repository includes integrations for several common tools (examples: `kind`, `kubectl`, `helm`, `talosctl`) — each tool exposes the same set of common subcommands described below.

### Common subcommands

- `list`
	- Lists all versions of the tool that are currently installed under the configured `vrs-path`.
	- Marks the version currently in use with an asterisk (`*`).

- `list-remote`
	- Lists remote versions available upstream (GitHub releases by default), sorted by semantic version.
	- Flags: `--devel` include pre-release versions (alpha/beta/rc), `-l, --limit` limit number of versions shown, `-f, --force` force refresh of the remote cache.

- `install <version>`
	- Downloads and installs the specified version for the current OS/ARCH.
	- Depending on the tool configuration the install may use GitHub releases or a direct download URL. The downloaded binary is stored under `vrs-path/<tool>/<tool>-<version>`.
	- After installing, run `use <version>` to activate it.

- `use <version>`
	- Makes the specified version the active one by creating (or replacing) a symlink named after the tool in the configured `bin-path` that points to the chosen `vrs-path` binary (e.g. `bin/<tool>` -> `vrs-path/<tool>/<tool>-<version>`).

---

## A note on Authentication

This tool makes use of the GitHub APIs.
You can use it without setting up anything if your use is infrequent.

But, if you plan on downloading a lot of versions (or frequently check which versions are available),
I strongly recommend setting up a `GITHUB_TOKEN` to get around the API rate-limiting.

Unauthenticated calls to the APIs are limited to 60 per hour.
Using a token you can get up to 5,000 per hour.

During development I observed, for example, that the `list-remote` can make up to at least 12 API calls to retrieve the whole list.
Do it 5 times and you're done.

### How to

To create a token:
1. go to https://github.com/settings/tokens/new
2. add a note
3. set the expiration you like
4. select scope: `repo > public_repo`.
5. scroll to the bottom of the page and click "Generate token"
6. copy the token that will appear

Now, up to you where you save it.
In the repo you have a `.env.template` file you can use: put your token there and rename the file to `.env` (don't worry, it's in the .gitignore).
Then, when you need to run the tool do:

```sh
source .env
```

Now run the tool and you'll have the limit upped to 5,000 calls an hour.

---

See the [docs folder](./docs/) for more information on the subcommands.
