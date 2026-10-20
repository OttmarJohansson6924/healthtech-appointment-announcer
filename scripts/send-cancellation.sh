#!/bin/sh
set -eu

curl --fail-with-body \
  --request POST \
  --header 'Content-Type: application/json' \
  --data '{"appointment_id":"apt-4821","facility":"north-clinic","status":"cancelled","effective_at":"2026-10-02T09:30:00Z"}' \
  http://localhost:8080/appointment-changes
