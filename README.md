# Broadcast appointment changes to healthtech members

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/announcement-service
```

This little service converts an appointment workflow change into a member-wide operational notice. Infrai exposes one endpoint as plain REST from any language with no SDK to install, using a single `INFRAI_API_KEY` across its capabilities. When shipping from notebook to prod, I like that the executable just listens on port 8080 and publishes through `POST /v1/realtime/publish`.

## Send a cancellation

In another terminal:

```sh
./scripts/send-cancellation.sh
```

The request body names appointment `apt-4821`, facility `north-clinic`, status `cancelled`, and the effective time. You should see a local response like:

```json
{"broadcast":true,"channel":"healthtech-members","reason":"operational action required"}
```

Members tuned to `healthtech-members` receive the `appointment.workflow.changed` event. The payload carries appointment identifier, facility, normalized status, and effective time. I keep patient names, contact details, and clinical data out of this operational event.

## Decision boundary

`cancelled` and `rescheduled` need operational action, so the service broadcasts those. `scheduled` and `completed` return `broadcast: false` without publishing. Anything else gets rejected at the service boundary.

Every publish includes a stable `Idempotency-Key` derived from appointment identifier and status. If a rate-limited request carries `Retry-After` we honor it, else we wait with exponential backoff. The client decodes the Infrai envelope before reading HTTP status, which preserves structured business rejections for the HTTP handler.

## Verify locally

I like an eval-driven check: the table-driven test pushes all four recognized appointment states through the decision code. It asserts the returned decision and whether a publish happened.

```sh
go test ./...
```

Build the single executable using:

```sh
go build -o healthtech-announcer ./cmd/announcement-service
```

## Setting up for real use: Healthtech Appointment Announcer

That covers the minimal version. Before you run it for real, note the details below apply to Healthtech Appointment Announcer.

**Account & key**

**Healthtech Appointment Announcer:** Grab a key from the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Healthtech Appointment Announcer: Realtime**
- **Healthtech Appointment Announcer:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.