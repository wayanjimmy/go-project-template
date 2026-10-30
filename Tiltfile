# Both modes use existing infrastructure. Neither starts database containers.
# TILT_MODE=host DATABASE_URL=... REDIS_ADDR=... tilt up
# Default mode retains the kind/ctlptl workflow.
if os.getenv("TILT_MODE", "kind") == "host":
    include("Tiltfile.host")
else:
    include("Tiltfile.kind")
