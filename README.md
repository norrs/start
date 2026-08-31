# Start

Start is a self-hosted internal launchpad for links, tools, and resources. It uses Vue 3, Gin, SQLite, Goose, sqlc, OIDC authentication over the OAuth 2.0 authorization-code flow, and a keyboard-first interface.

<!-- release:start -->
Latest release: [v0.0.1](https://github.com/norrs/start/releases/tag/v0.0.1)

Published multi-architecture container tags: `ghcr.io/norrs/start:0.0.1`,
`ghcr.io/norrs/start:0.0`, `ghcr.io/norrs/start:0`, and `ghcr.io/norrs/start:latest`.
<!-- release:end -->

## Quick start with the latest release

The production Compose file follows `ghcr.io/norrs/start:latest`. Download it,
create the persistent and secret directories, and copy the environment
template:

```sh
sudo install -d -m 0755 /opt/start
sudo install -d -o 65532 -g 65532 -m 0750 /opt/start/data
sudo install -d -m 0700 /opt/start/secrets
sudo wget -O /opt/start/docker-compose.yaml \
  https://raw.githubusercontent.com/norrs/start/main/deploy/quickstart/docker-compose.yaml
sudo wget -O /opt/start/start.env.example \
  https://raw.githubusercontent.com/norrs/start/main/deploy/quickstart/start.env.example
sudo wget -O /etc/systemd/system/start.service \
  https://raw.githubusercontent.com/norrs/start/main/deploy/quickstart/start.service
sudo cp /opt/start/start.env.example /opt/start/.env
sudo install -m 0600 /dev/null /opt/start/secrets/oidc_client_secret
sudoedit /opt/start/.env
sudoedit /opt/start/secrets/oidc_client_secret
sudo systemctl daemon-reload
sudo systemctl enable --now start
```

Register `<STARTPAGE_BASE_URL>/auth/callback` with the OIDC provider before
starting the service. The default loopback bind is intended for a local TLS
reverse proxy; change `START_BIND_ADDRESS` only when the network exposure is
intentional. You can also run the downloaded file without systemd:

```sh
cd /opt/start
sudo docker compose up --detach
```

For a loopback-only evaluation without an identity provider, download the
separate development file. `STARTPAGE_DEV_MODE` bypasses OIDC and signs requests
in as the configured development user, so never expose this instance to a
network:

```sh
mkdir start-dev && cd start-dev
mkdir data
sudo chown 65532:65532 data
wget https://raw.githubusercontent.com/norrs/start/main/deploy/quickstart/docker-compose.dev.yaml
docker compose -f docker-compose.dev.yaml up --detach
```

## Releases

Maintainers create a release from **Actions → Release → Run workflow** on the
`main` branch. `auto` applies Conventional Commits: `feat:` selects a minor
bump, a `!` or `BREAKING CHANGE:` selects major, and all other releasable
commits select patch. The workflow can also be given an explicit bump.

The equivalent GitHub CLI command is:

```sh
gh workflow run release.yml --ref main -f bump=auto
```

Every release reruns `//:all_tests`, publishes one amd64/arm64 image under the
full version, major/minor, major, and `latest` tags, and creates a GitHub
release. Periods are valid in OCI image tags, so a version such as `1.4` does
not need to become `1-4`. Add `docs/releases/v<version>.md` before releasing to
use curated notes; otherwise GitHub generates the notes. See
[`docs/releases/README.md`](docs/releases/README.md) for the complete policy.

After publishing, the workflow commits the latest release and image tags near
the top of this README. The repository must allow `github-actions[bot]` to push
that metadata commit to `main` (directly or through an appropriate branch-rule
bypass), and the GitHub Actions workflow token needs read/write permission.
The first GHCR package may also need to be made public in the repository or
organization package settings.

## Develop and verify

```sh
bazel build //cmd/start:start //web:dist
bazel test //:all_tests
```

`//:all_tests` includes Go unit tests, Vue/Vitest component tests, TypeScript
checking, and the Playwright UI suite. To run only the browser journeys:

```sh
bazel test //web:e2e_test
```

To watch Chromium perform the tests in a local graphical session, use the
headed Bazel configuration:

```sh
bazelisk test --config=e2e-headed //web:e2e_test
```

This forces a fresh run, streams the test names, passes through the desktop
display, and slows browser actions slightly.
It inherits `DISPLAY` and `XAUTHORITY`, so run it from a terminal inside the
graphical session. To change or disable the delay, append for example
`--test_env=START_E2E_SLOW_MO=250` or `--test_env=START_E2E_SLOW_MO=0`.
On a headless machine or an SSH session without display forwarding, use the
normal `bazelisk test //web:e2e_test` command instead.

For Playwright's interactive test explorer, run:

```sh
bazelisk test --config=e2e-ui //web:e2e_test
```

This launches Playwright UI in Bazel-managed Chromium rather than handing the
URL to the system's default browser. It uses the same application server,
temporary SQLite database, seed file, and test configuration as the automated
target. Select or rerun tests from the explorer and press `Ctrl+C` in the
terminal when finished. This also requires a graphical session. Keep this as a
`bazelisk test` command:
running a generated JavaScript test executable with `bazelisk run` bypasses the
test runfiles layout and can make Playwright resolve duplicate module instances.

The explorer is organized as product journeys rather than implementation
details. Expand **Start product journeys**, choose a story such as **manage a
link through its complete lifecycle**, and run it. After it runs, expand the
named steps to follow actions such as opening settings, editing a link, and
restoring history. The story definitions live in
`web/e2e/specs/start.spec.ts`; reusable HTML navigation and accessible
selectors live in `web/e2e/support/start-page.ts`; API-only setup and cleanup
live in `web/e2e/support/catalog-api.ts`. Stop and restart the UI command after
changing these files because the active Bazel test sandbox keeps its original
runfiles snapshot.

The Playwright target downloads Bazel-managed Chromium runtimes and
starts the real Go application with a fresh temporary SQLite database seeded
from `links.yaml`. It covers search and keyboard navigation, view and theme
preferences, hide/show, link CRUD and icon previews, per-link and selective
history restore, non-admin controls, logout, health/metrics, and custom 404s.
Failed runs retain Playwright traces, screenshots, and video in the Bazel test
outputs.

### Playwright failure recordings

The shared Playwright configuration uses `video: 'retain-on-failure'`,
`screenshot: 'only-on-failure'`, and `trace: 'retain-on-failure'`. Successful
journeys discard their media. Failed attempts are available below:

```text
bazel-testlogs/web/e2e_test/test.outputs/playwright/<journey>/video.webm
bazel-testlogs/web/e2e_test/test.outputs/playwright/<journey>/test-failed-1.png
bazel-testlogs/web/e2e_test/test.outputs/playwright/<journey>/trace.zip
```

List the exact artifact paths after a failure with:

```sh
find -L bazel-testlogs/web/e2e_test/test.outputs/playwright \
  -type f \( -name '*.webm' -o -name '*.png' -o -name 'trace.zip' \)
```

`video.webm` is a normal WebM file: open it in Firefox or Chromium, or play it
with a desktop player such as VLC or mpv. Video is best for seeing timing,
animation, and what a user saw. For diagnosis, the trace is usually more
valuable because it includes the action timeline, DOM snapshots, network
requests, console messages, and attachments. Open a downloaded or local trace
through Bazel from the repository root:

```sh
bazelisk run //web:show_trace -- /absolute/path/to/trace.zip
```

The path may also be relative to the repository root. The viewer runs in
Bazel-managed Chromium and does not require a global Playwright or browser
installation.

To temporarily retain recordings for successful journeys as well as failures,
use the recording configuration:

```sh
bazelisk test --config=e2e-record //web:e2e_test
```

This sets `START_E2E_RECORD_ALL=true` inside the Bazel test sandbox, forces a
fresh run, records video and trace for every journey, and takes an end-of-test
screenshot. The normal, UI, headed, and CI configurations remain failure-only.
Playwright 1.59's native video overlays show the current product-level test
step at the top and browser actions at the bottom. Recorded runs use a longer
overlay duration and a default 500 ms action delay so the story remains
readable during playback.
The equivalent one-off command is:

```sh
bazelisk test //web:e2e_test --nocache_test_results \
  --test_env=START_E2E_RECORD_ALL=true
```

Setting only a shell environment variable is insufficient because Bazel does
not pass arbitrary host variables into a hermetic test; use the named
configuration or `--test_env`.

CI runs all Bazel tests headlessly and retries a failed Playwright journey once.
The workflow uploads the Playwright output directory even when a retry passes,
so evidence from a flaky first attempt is not lost. On GitHub, open the workflow
run, download the **playwright-failure-artifacts** artifact, unzip it, and play
the WebM directly or open `trace.zip` with the command above. CI retains these
artifacts for 14 days. Traces and videos can contain page content and URLs, so
do not publish them from workflows that exercise sensitive production data.

For the quickest local test, build the frontend and backend and start them
together in development identity mode:

```sh
bazel run //:dev
```

Open `http://127.0.0.1:8080`. The target keeps its local SQLite database in
`.start-dev/start.db`, which is ignored by Git. Delete that directory when you
want a fresh import from `links.yaml`, or override any normal setting, for
example:

```sh
STARTPAGE_LISTEN_ADDR=127.0.0.1:18080 \
STARTPAGE_BASE_URL=http://127.0.0.1:18080 \
bazel run //:dev
```

For direct frontend work, install the pinned pnpm version from `web/package.json`, then run `pnpm dev` in `web/`. Run the API in development identity mode with:

```sh
STARTPAGE_DEV_MODE=true \
STARTPAGE_DATABASE_PATH=/tmp/start.db \
STARTPAGE_STATIC_DIR="$PWD/bazel-bin/web/dist" \
bazel run //cmd/start:start
```

The normal server startup imports `links.yaml` only when links, categories, and
owner groups are all empty. Later YAML changes never silently overwrite
database-managed data.

To intentionally merge a YAML catalog into a populated database, use the
explicit importer:

```sh
bazelisk run //:import-links -- \
  --database .start-dev/start.db \
  --file links.yaml
```

The importer uses slugs as stable identities. It upserts categories and owner
groups, creates new links, updates matching links, restores matching
soft-deleted links, and leaves records omitted from the YAML untouched. Link
updates and restores appear in the admin history and can be rolled back there.
Repeating an unchanged import is a no-op.

System administrators can perform the same merge from the web interface using
**Import YAML** in the header, or the <kbd>I</kbd> shortcut. The dialog previews
the selected `.yaml`/`.yml` file and requires an explicit acknowledgement
before importing. Web uploads are limited to 1 MiB, validated in full, applied
atomically, and attributed to the administrator in link history.

For a deployed database, stop the Start container (or otherwise ensure you have
the correct writable database path), take a backup, and pass that path to the
same target. For example:

```sh
cp /srv/start/data/start.db /srv/start/data/start.db.before-import
bazelisk run //:import-links -- \
  --database /srv/start/data/start.db \
  --file /path/to/private-links.yaml
```

Host-specific catalogs should not be committed. This checkout ignores files
matching `links.*.yaml`; a private catalog can be loaded into the local
development database with:

```sh
bazelisk run //:import-links -- \
  --database .start-dev/start.db \
  --file links.private.yaml
```

## Container image

```sh
bazel build //deploy:image          # linux/amd64 OCI layout
bazel build //deploy:image_index    # linux/amd64 + linux/arm64 index
bazel run //deploy:load             # load localhost/start:dev into Docker/Podman
```

The image runs as UID/GID `65532`, uses a read-only root filesystem, listens on port 8080, and writes only to `/data` and `/tmp`. See `deploy/compose.example.yaml` for the complete runtime contract. The persistent host data directory must be writable by UID 65532.

To build on your local machine and load the image into Docker on another
amd64 host, first let Bazel build and load the image locally:

```sh
cd ~/dev/start
bazel run //deploy:load
docker image inspect localhost/start:dev
```

Then stream the Docker archive over SSH. This avoids leaving an image archive
on either host and preserves the `localhost/start:dev` tag expected by the
deployment. Replace `remote-host` with your target host:

```sh
docker save localhost/start:dev | gzip | ssh remote-host 'gzip -d | docker load'
ssh remote-host 'docker image inspect localhost/start:dev'
```

If remote Docker requires an interactive `sudo` password, first copy a
compressed archive and then allocate a terminal for the remote load:

```sh
cd ~/dev/start
bazel run //deploy:load
docker save localhost/start:dev | gzip > /tmp/start-dev.tar.gz
scp /tmp/start-dev.tar.gz remote-host:start-dev.tar.gz
ssh -t remote-host \
  'gzip -dc "$HOME/start-dev.tar.gz" | sudo docker load'
```

Verify the loaded image, then remove the temporary local and remote archives:

```sh
ssh -t remote-host 'sudo docker image inspect localhost/start:dev'
ssh remote-host 'rm "$HOME/start-dev.tar.gz"'
rm /tmp/start-dev.tar.gz
```

After the remote inspection succeeds, deploy the service using
`deploy/compose.example.yaml` or your normal deployment automation.

## Identity deployment

Start is a native OpenID Connect client and uses the OAuth 2.0 authorization
code flow. The identity service must provide OIDC discovery and ID tokens; a
generic OAuth-only provider is not sufficient. For a deployment at
`https://start.example.com` using Tinyauth at `https://auth.example.com`,
register the callback
`https://start.example.com/auth/callback`. Tinyauth may delegate login to
Pocket ID. Do not additionally wrap Start in Tinyauth forward-auth.

Logout revokes the Start session and navigates to the trusted browser-visible
route configured by `STARTPAGE_OIDC_LOGOUT_URL`, for example
`https://auth.example.com/logout`. When Tinyauth uses Pocket ID upstream,
configure Tinyauth to perform Pocket ID RP-initiated logout and allow
Tinyauth's callback, for example
`https://auth.example.com/api/user/logout/callback`, in the Pocket ID client.

Required claims are `sub`, `email`, profile/name, and `groups`. `start-admins` grants bootstrap system administration. Other exact Pocket ID group values are mapped to local roles, teams, or owner-group grants in SQLite; unmapped groups grant nothing.

Back up SQLite with its online backup mechanism or a brief write pause. Preserve `start.db`, `start.db-wal`, and `start.db-shm` together if copying files directly. Link deletion is soft and every mutation records a restorable revision plus an audit event. Authorized administrators can open the global History UI (or press `G`), select one historical snapshot for each of any subset of links, and restore that subset atomically without changing other links.
