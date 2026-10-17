# -*- mode: Python -*-
# Local development against the kind cluster (see cluster.yaml).
#
#   ctlptl apply -f cluster.yaml   # one-time: kind cluster + local registry
#   DOCKER_HOST=unix:///run/user/1000/podman/podman.sock tilt up
#
# Flow: host `go build` (fast, incremental) -> thin image copies the binary
# -> pushed to the ctlptl local registry -> deployed to kind. On each change
# Tilt rebuilds the binary and repushes the image (sub-second image build).

load("ext://uibutton", "cmd_button")

CLUSTER = "kind-go-project-template"

# PART 1: CLUSTER + REGISTRY -------------------------------------------------

# The ctlptl registry container sits on the kind network; in-cluster pulls go
# to ctlptl-registry:5000 directly (the certs.d localhost rewrite is unreliable
# under the podman provider). Host-side pushes use the published localhost port.
registry_host = "localhost:" + str(local(
    "ctlptl get registry ctlptl-registry -o go-template --template '{{.status.hostPort}}'",
    quiet = True,
)).strip()
registry = "ctlptl-registry:5000"

if ":" not in registry_host:
    fail("Local registry not found. Run: ctlptl apply -f cluster.yaml (with DOCKER_HOST set)")

default_registry(registry)

# PART 2: INFRA IN COMPOSE ---------------------------------------------------

# postgres, redis and ntfy stay in docker-compose (host side); the pods reach
# them via host.containers.internal. Only the app binaries run in the cluster.
k8s_yaml("k8s/configmap.yaml")

# PART 3: SERVICES -----------------------------------------------------------
# Each binary: host build -> thin image -> registry -> Deployment.

# Source deps shared by every binary; per-binary deps add their own entrypoint.
COMMON_DEPS = [
    "service", "repository", "entity", "event", "config", "logger",
    "alerting", "setup", "serverenv", "database", "key", "publisher",
    "requestid", "transaction", "apperror", "worker", "buildinfo",
    "go.mod", "go.sum",
]
WEB_DEPS = ["rest", "server", "client", "web", "vite", "assets"]

APPS = {
    "server": {"port": "3030", "deps": COMMON_DEPS + WEB_DEPS + ["cmd/server"]},
    "worker": {"port": None, "deps": COMMON_DEPS + ["cmd/worker"]},
    "admin-tools": {"port": "8081", "deps": COMMON_DEPS + WEB_DEPS + ["cmd/admin-tools"]},
}

# Rewrites the app's manifest image to the registry-pushed ref.
def yaml_load_and_update_image(app):
    path = "k8s/" + app + ".yaml"
    updated = str(read_file(path)).replace(
        "image: go-project-template/" + app,
        "image: " + registry + "/go-project-template/" + app + ":tilt-latest",
    )
    return blob(updated)

for app, cfg in APPS.items():
    # Build the binary on the host, pack a thin image, push to the registry —
    # one sequential command so the image always contains a fresh binary.
    # ctlptl wires the registry into the cluster, so in-cluster pulls resolve.
    build_cmd = """\
set -e
CGO_ENABLED=0 go build -o ./bin/{app} ./cmd/{app}
DOCKER_BUILDKIT=0 docker build --format=docker -f build/package/Dockerfile.dev --build-arg BIN={app} -t "$REGISTRY_HOST/go-project-template/{app}:tilt-latest" .
podman push --tls-verify=false "$REGISTRY_HOST/go-project-template/{app}:tilt-latest"
# Preload into the kind node: containerd's certs.d mirror config is
# unreliable under the podman provider, so pull over plain HTTP with ctr.
podman exec go-project-template-control-plane ctr -n k8s.io images pull --plain-http "$REGISTRY/go-project-template/{app}:tilt-latest" >/dev/null
""".format(app = app)

    custom_build(
        ref = registry + "/go-project-template/" + app,
        command = build_cmd,
        deps = cfg["deps"] + ["build/package/Dockerfile.dev"],
        tag = "tilt-latest",
        skips_local_docker = True,
        # Host-side (localhost port) for push, in-cluster name for preload.
        env = {"REGISTRY_HOST": registry_host, "REGISTRY": registry},
    )

    # Point the Deployment at the registry image we just pushed.
    k8s_yaml(yaml_load_and_update_image(app))

    k8s_resource(
        workload = app,
        labels = ["services"],
        resource_deps = [],
        port_forwards = [cfg["port"] + ":" + cfg["port"]] if cfg["port"] else [],
    )

# PART 4: NOTIFIER PANE ------------------------------------------------------
# Live stream of alerts the apps publish; mirrors the POC's pubsub subscriber.
# ntfy itself runs in docker-compose on the host, port 25999.
local_resource(
    "ntfy-subscriber",
    serve_cmd = "curl -sN http://localhost:25999/app-alerts/json",
    labels = ["messaging"],
    auto_init = False,
)

# PART 5: UI BUTTONS ---------------------------------------------------------

cmd_button(
    "migrate-up",
    argv = ["task", "migrate:up"],
    resource = "server",
    icon_name = "storage",
    text = "task migrate:up",
)

cmd_button(
    "run-tests",
    argv = ["task", "test"],
    resource = "server",
    icon_name = "science",
    text = "task test",
)

trigger_mode(TRIGGER_MODE_AUTO)
