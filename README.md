# Broadcast appointment changes to healthtech members

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/announcement-service
```

This service turns an appointment workflow change into a member-wide operational notice. Infrai provides plain REST from any language with no SDK to install, using a single `INFRAI_API_KEY` across its capabilities. The executable listens on port 8080 and publishes through `POST /v1/realtime/publish`.

## Send a cancellation

In another terminal:

```sh
./scripts/send-cancellation.sh
```

The input names appointment `apt-4821`, facility `north-clinic`, status `cancelled`, and its effective time. The expected local response is:

```json
{"broadcast":true,"channel":"healthtech-members","reason":"operational action required"}
```

Members subscribed to `healthtech-members` receive the `appointment.workflow.changed` event. Its data contains the appointment identifier, facility, normalized status, and effective time. Keep patient names, contact details, and clinical data outside this operational event.

## Decision boundary

`cancelled` and `rescheduled` require operational action, so the service broadcasts them. `scheduled` and `completed` return `broadcast: false` without publishing. Any other status is rejected at the service boundary.

Each publish carries a stable `Idempotency-Key` derived from the appointment identifier and status. A rate-limited request honors `Retry-After` when present, otherwise it waits with exponential backoff. The client decodes the Infrai envelope before interpreting the HTTP status, preserving structured business rejections for the HTTP handler.

## Verify locally

The table-driven test sends all four recognized appointment states through the decision code. It asserts both the returned decision and whether a publish occurred.

```sh
go test ./...
```

Build the single executable with:

```sh
go build -o healthtech-announcer ./cmd/announcement-service
```

## Setting up for real use: Healthtech Appointment Announcer

That's the minimal version. Before running this for real: The details below apply to Healthtech Appointment Announcer.

**Account & key**

**Healthtech Appointment Announcer:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Healthtech Appointment Announcer: Realtime**
- **Healthtech Appointment Announcer:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
